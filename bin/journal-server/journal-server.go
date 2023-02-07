package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
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
)

// DraftTimeout measures how long it takes for an unsaved draft to get added to the journal.
const DraftTimeout time.Duration = 2 * time.Hour

// DraftExpireInterval is the interval at which the application checks if any unsaved drafts have expired and should get auto-added
const DraftExpireInterval time.Duration = 15 * time.Minute

type draftEntry struct {
	LastEdit time.Time
	Expires  time.Time
	Body     string
}

var (
	draftsMutex sync.Mutex
	drafts      map[string]draftEntry
)

func init() {
	flag.Parse()
	drafts = make(map[string]draftEntry)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
func run() error {
	r := mux.NewRouter()
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
		err := saveJournalEntry(entry.LastEdit, entry.Body, false)
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
				err := saveJournalEntry(entry.LastEdit, entry.Body, false)
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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	indexData := struct {
	}{}

	executeTemplate(index, indexData, w, r)
}

func WriterHandler(w http.ResponseWriter, r *http.Request) {
	getv := r.URL.Query()

	getv.Del("success")
	getv.Del("failure")

	homeData := struct {
		Success, Failure bool
		Callback         string
	}{
		r.URL.Query().Get("success") != "",
		r.URL.Query().Get("failure") != "",
		"journal?" + getv.Encode(),
	}

	executeTemplate(editor, homeData, w, r)
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

func saveJournalEntry(timestamp time.Time, contents string, starred bool) error {
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

	// Remove carriage returns entirely. Why? Because it fits my use case, and because sod MS-DOS.
	body = strings.Replace(body, "\r", "", -1)
	for len(body) > 0 && body[len(body)-1] == '\n' {
		body = body[0 : len(body)-1]
	}

	err := saveJournalEntry(timestamp, body, starred)
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
	getv.Set("success", "1")

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

	draftsMutex.Lock()
	defer draftsMutex.Unlock()
	if post_body == "" {
		delete(drafts, draft_id)
	} else {
		drafts[draft_id] = draftEntry{
			LastEdit: time.Now(),
			Expires:  time.Now().Add(DraftTimeout),
			Body:     post_body,
		}
	}

	rv := struct {
		OK      int    `json:"ok"`
		Message string `json:"_"`
		DraftID string `json:"draft_id"`
	}{1, "Draft saved", draft_id}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(rv)
}
