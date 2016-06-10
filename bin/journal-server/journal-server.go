package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/thijzert/go-journal"
	"log"
	"net/http"
	"strings"
)

// This package requires go-bindata (github.com/jteeuwen/go-bindata) to build
//go:generate go-bindata -o assets.go -pkg main assets/...
// For development purposes, this command is much more convenient:
//
//     go-bindata -debug -o assets.go -pkg main assets/...

var (
	listen       = flag.String("listen", ":8848", "Listen on this host/port")
	journal_file = flag.String("journal_file", "journal.txt", "Use this file for Journal storage")
)

func init() {
	flag.Parse()
}

func main() {
	fmt.Printf("Listening on '%s'; storing everything in '%s'.\n", *listen, *journal_file)

	r := mux.NewRouter()
	r.Methods("GET").Path("/").HandlerFunc(HomeHandler)
	r.Methods("POST").Path("/").HandlerFunc(SaveHandler)
	r.PathPrefix("/assets/").HandlerFunc(AssetHandler)

	log.Printf("Web frontend starting on %s...\n", *listen)
	log.Fatal(http.ListenAndServe(*listen, r))
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	getv := r.URL.Query()

	homeData := struct {
		Success, Failure bool
	}{
		getv.Get("success") != "",
		getv.Get("failure") != ""}

	executeTemplate(index, homeData, w, r)
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

	w.Header().Set("Location", "/?success=1")
	w.WriteHeader(http.StatusFound)
}
