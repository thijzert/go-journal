package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/thijzert/go-journal"
	"github.com/thijzert/go-journal/bin/journal-server/secretbookmark"
	"log"
	"net/http"
	"strings"
	"time"
)

// This package requires go-bindata (github.com/jteeuwen/go-bindata) to build
//go:generate go-bindata -o assets.go -pkg main assets/...
// For development purposes, this command is much more convenient:
//
//     go-bindata -debug -o assets.go -pkg main assets/...

var (
	listen           = flag.String("listen", ":8848", "Listen on this host/port")
	journal_file     = flag.String("journal_file", "journal.txt", "Use this file for Journal storage")
	password_file    = flag.String("password_file", ".htpasswd", "File containing passwords")
	secret_parameter = flag.String("secret_parameter", "apikey", "Parameter name containing the API key")
)

func init() {
	flag.Parse()
}

func main() {
	r := mux.NewRouter()
	r.Methods("GET").Path("/journal").HandlerFunc(RequireLoggedIn(WriterHandler))
	r.Methods("POST").Path("/journal").HandlerFunc(RequireLoggedIn(SaveHandler))
	r.Path("/tie/{date}.svg").HandlerFunc(TieHandler)
	r.Path("/bwv").HandlerFunc(BWVHandler)
	r.PathPrefix("/assets/").HandlerFunc(AssetHandler)

	p := secretbookmark.NewHandler(r, *secret_parameter, *password_file)

	log.Printf("Listening on '%s'; storing everything in '%s'.\n", *listen, *journal_file)
	log.Fatal(http.ListenAndServe(*listen, p))
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
		"journal?" + getv.Encode()}

	executeTemplate(editor, homeData, w, r)
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

	e := &journal.Entry{
		Date:     timestamp,
		Starred:  starred,
		Contents: body}

	err := journal.Add(*journal_file, e)
	if err != nil {
		errorHandler(err, w, r)
		return
	}

	getv := r.URL.Query()
	getv.Del("failure")
	getv.Set("success", "1")

	w.Header().Set("Location", "journal?"+getv.Encode())
	w.WriteHeader(http.StatusFound)
}

func TieHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	date, err := time.Parse("2006-01-02", vars["date"])

	if err != nil {
		w.Header()["Content-Type"] = []string{"text/plain"}
		w.WriteHeader(404)
		w.Write([]byte("No tie was found for that day.\n\nLive a little; wear a t-shirt.\n"))
	}

	w.Header()["Content-Type"] = []string{"image/svg"}

	tieData := struct {
		Colour string
	}{"teal"}

	if date.Year() == 1988 {
		tieData.Colour = "pink"
	}

	executeTemplate(tie, tieData, w, r)
}
