package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/thijzert/go-journal"
	"github.com/thijzert/go-journal/bin/journal-server/secretbookmark"
)

var (
	listen           = flag.String("listen", ":8848", "Listen on this host/port")
	journal_file     = flag.String("journal_file", "journal.txt", "Use this file for Journal storage")
	password_file    = flag.String("password_file", ".htpasswd", "File containing passwords")
	secret_parameter = flag.String("secret_parameter", "apikey", "Parameter name containing the API key")
	attachments_dir  = flag.String("attachments_dir", "", "Directory for storing attached files")
	projects_dir     = flag.String("projects_dir", "", "Directory with project log files")
)

// DraftTimeout measures how long it takes for an unsaved draft to get added to the journal.
const DraftTimeout time.Duration = 2 * time.Hour

// DraftExpireInterval is the interval at which the application checks if any unsaved drafts have expired and should get auto-added
const DraftExpireInterval time.Duration = 15 * time.Minute

type draftEntry struct {
	LastEdit time.Time
	Expires  time.Time
	Body     string
	Project  string
}

var (
	draftsMutex sync.Mutex
	drafts      map[string]draftEntry
)

// AttachmentTimeout measures how long an attachment should remain cached
const AttachmentTimeout time.Duration = 30 * time.Minute

// AttachmentTimeout measures how long an attachment should remain cached
const AttachmentPurgeInterval time.Duration = 5 * time.Minute

type attachmentEntry struct {
	PurgeAt time.Time
	Buf     []byte
}

var (
	attachmentMutex sync.Mutex
	attachments     map[string]attachmentEntry
)

func init() {
	flag.Parse()
	drafts = make(map[string]draftEntry)
	attachments = make(map[string]attachmentEntry)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
func run() error {
	r := mux.NewRouter()
	r.Methods("POST").Path("/journal/attachment").HandlerFunc(RequireLoggedIn(FileUploadHandler))
	r.Methods("POST").Path("/journal/draft").HandlerFunc(RequireLoggedIn(SaveDraftHandler))
	r.Methods("GET").Path("/journal").HandlerFunc(RequireLoggedIn(WriterHandler))
	r.Methods("POST").Path("/journal").HandlerFunc(RequireLoggedIn(SaveHandler))
	r.Methods("GET").Path("/daily").HandlerFunc(RequireLoggedIn(DailyHandler))
	r.Methods("POST").Path("/daily").HandlerFunc(RequireLoggedIn(SaveHandler))
	r.Path("/tie").HandlerFunc(AllTiesHandler)
	r.Path("/tie/{date}.svg").HandlerFunc(TieHandler)
	r.Path("/bwv").HandlerFunc(BWVHandler)
	r.PathPrefix("/assets/").HandlerFunc(AssetHandler)
	r.Path("/").HandlerFunc(IndexHandler)

	p := secretbookmark.New(*secret_parameter, *password_file)
	r.Use(p.Middleware)

	defer onShutdown()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go autoAddDrafts(ctx)
	go autoPurgeAttachments(ctx)

	var lc net.ListenConfig
	var err error
	var l net.Listener

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		l.Close()
	}()

	l, err = lc.Listen(ctx, "tcp", *listen)
	if err != nil {
		return err
	}
	log.Printf("Listening on '%s'; storing everything in '%s'.\n", *listen, *journal_file)

	err = http.Serve(l, r)

	if err == nil || errors.Is(err, net.ErrClosed) {
		err = ctx.Err()
	}
	if err == nil || errors.Is(err, context.Canceled) {
		err = nil
	}
	return err
}

func onShutdown() {
	log.Printf("Shutting down")

	// Save all pending drafts.
	draftsMutex.Lock()
	for draft_id, entry := range drafts {
		log.Printf("Add draft ID %s to journal: last saved at %s", draft_id, entry.LastEdit)
		err := saveJournalEntry(entry.LastEdit, entry.Body, entry.Project, false)
		if err != nil {
			log.Printf("Error saving journal entry: %v", err)
		}
	}
	draftsMutex.Unlock()

	log.Printf("Shutdown complete")
}

func autoAddDrafts(ctx context.Context) {
	ticker := time.NewTicker(DraftExpireInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			toDelete := []string{}
			draftsMutex.Lock()
			for draft_id, entry := range drafts {
				if entry.Expires.After(time.Now()) {
					continue
				}

				log.Printf("Draft ID %s expired at %s; saving it to journal", draft_id, entry.Expires)
				err := saveJournalEntry(entry.LastEdit, entry.Body, entry.Project, false)
				if err != nil {
					log.Printf("Error saving journal entry: %v", err)
				} else {
					toDelete = append(toDelete, draft_id)
				}
			}
			for _, draft_id := range toDelete {
				delete(drafts, draft_id)
			}
			draftsMutex.Unlock()
		}
	}
}

func autoPurgeAttachments(ctx context.Context) {
	ticker := time.NewTicker(AttachmentPurgeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			toDelete := []string{}
			attachmentMutex.Lock()
			for att_hash, entry := range attachments {
				if entry.PurgeAt.After(time.Now()) {
					continue
				}
				toDelete = append(toDelete, att_hash)
			}
			for _, att_hash := range toDelete {
				log.Printf("Deleting attachment with hash '%s'", att_hash)
				delete(attachments, att_hash)
			}
			attachmentMutex.Unlock()
		}
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	indexData := struct {
	}{}

	executeTemplate(index, indexData, w, r)
}

func listProjects(ctx context.Context) ([]string, error) {
	if *projects_dir == "" {
		return nil, nil
	}

	d, err := os.Open(*projects_dir)
	if err != nil {
		return nil, err
	}

	fis, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	var rv []string
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		rv = append(rv, fi.Name())
	}

	sort.Strings(rv)

	return rv, nil
}

func WriterHandler(w http.ResponseWriter, r *http.Request) {
	getv := r.URL.Query()

	getv.Del("success")
	getv.Del("failure")

	projects, _ := listProjects(r.Context())

	pageData := struct {
		Success, Failure bool
		Callback         string
		CanAttachFiles   bool
		Projects         []string
	}{
		r.URL.Query().Get("success") != "",
		r.URL.Query().Get("failure") != "",
		"journal?" + getv.Encode(),
		*attachments_dir != "",
		projects,
	}

	executeTemplate(editor, pageData, w, r)
}

func DailyHandler(w http.ResponseWriter, r *http.Request) {
	getv := r.URL.Query()

	getv.Del("success")
	getv.Del("failure")

	pageData := struct {
		Success, Failure bool
		Callback         string
	}{
		r.URL.Query().Get("success") != "",
		r.URL.Query().Get("failure") != "",
		"daily?" + getv.Encode(),
	}

	executeTemplate(daily, pageData, w, r)
}

func saveJournalEntry(timestamp time.Time, contents string, project string, starred bool) error {
	if project != "" && *projects_dir != "" {
		prf := path.Join(*projects_dir, strings.Replace(strings.Replace(project, "/", "", -1), "\\", "", -1))
		if f, err := os.OpenFile(prf, os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			fmt.Fprintf(f, "\n=== %s ===\n%s\n", timestamp.Format("2006-01-02"), contents)
			f.Close()
		}
	}
	if project != "" {
		contents = "@project " + formatProjectName(project) + "\n" + contents
	}

	e := &journal.Entry{
		Date:     timestamp,
		Starred:  starred,
		Contents: contents,
	}

	return journal.Add(*journal_file, e)
}

func SaveHandler(w http.ResponseWriter, r *http.Request) {
	timestamp := journal.SmartTime(r.PostFormValue("ts"))
	starred := r.PostFormValue("star") != ""
	body := r.PostFormValue("body")
	project := r.PostFormValue("project")

	// Remove carriage returns entirely. Why? Because it fits my use case, and because sod MS-DOS.
	body = strings.Replace(body, "\r", "", -1)
	for len(body) > 0 && body[len(body)-1] == '\n' {
		body = body[0 : len(body)-1]
	}

	var nonFatalError error

	if *attachments_dir != "" {
		attachmentMutex.Lock()
		defer attachmentMutex.Unlock()

		for att_hash, entry := range attachments {
			if r.PostFormValue("attachment-"+att_hash) == "" {
				continue
			}

			delete(attachments, att_hash)

			// Link the attachment in the post body
			body = fmt.Sprintf("%s\n@attachment %s", body, att_hash)

			f, err := os.Create(path.Join(*attachments_dir, att_hash))
			if err != nil {
				nonFatalError = err
				continue
			}
			f.Write(entry.Buf)
			f.Close()
		}
	}

	err := saveJournalEntry(timestamp, body, project, starred)
	if err != nil {
		errorHandler(err, w, r)
		return
	}

	if draft_id := r.PostFormValue("draft_id"); draft_id != "" {
		// We're saving this post - no need to keep the draft around
		draftsMutex.Lock()
		delete(drafts, draft_id)
		draftsMutex.Unlock()
	}

	getv := r.URL.Query()
	getv.Del("failure")
	getv.Del("success")
	if nonFatalError != nil {
		getv.Set("failure", "1")
	} else {
		getv.Set("success", "1")
	}

	w.Header().Set("Location", path.Base(r.URL.Path)+"?"+getv.Encode())
	w.WriteHeader(http.StatusFound)
}

func SaveDraftHandler(w http.ResponseWriter, r *http.Request) {
	draft_id := r.PostFormValue("draft_id")
	if len(draft_id) != 12 {
		buf := make([]byte, 6)
		_, err := rand.Read(buf)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}
		draft_id = hex.EncodeToString(buf)
	}

	post_body := r.PostFormValue("body")
	project := r.PostFormValue("project")

	draftsMutex.Lock()
	defer draftsMutex.Unlock()
	if post_body == "" {
		delete(drafts, draft_id)
	} else {
		drafts[draft_id] = draftEntry{
			LastEdit: time.Now(),
			Expires:  time.Now().Add(DraftTimeout),
			Body:     post_body,
			Project:  project,
		}
	}

	writeJSON(w, struct {
		OK      int    `json:"ok"`
		Message string `json:"_"`
		DraftID string `json:"draft_id"`
	}{1, "Draft saved", draft_id})
}

func FileUploadHandler(w http.ResponseWriter, r *http.Request) {
	if *attachments_dir == "" {
		writeJSONError(w, 503, 503, "This feature is not available")
		return
	}

	att_hash := r.URL.Query().Get("att_hash")
	if len(att_hash) != 64 {
		writeJSONError(w, 400, 400, "Invalid attachment hash")
		return
	}
	if _, err := hex.DecodeString(att_hash); err != nil {
		writeJSONError(w, 400, 400, "Invalid attachment hash")
		return
	}

	chunk, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading chunk: %v")
		writeJSONError(w, 400, 400, "Error reading chunk")
		return
	}

	attachmentMutex.Lock()

	e := attachments[att_hash]
	e.PurgeAt = time.Now().Add(AttachmentTimeout)
	e.Buf = append(e.Buf, chunk...)
	file_length := len(e.Buf)
	attachments[att_hash] = e

	attachmentMutex.Unlock()

	writeJSON(w, struct {
		OK      int    `json:"ok"`
		Message string `json:"_"`
		Length  int    `json:"file_length"`
	}{1, "Chunk saved", file_length})
}
