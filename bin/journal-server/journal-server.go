package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/mux"
	"github.com/thijzert/go-journal"
	"github.com/thijzert/go-journal/bin/journal-server/secretbookmark"
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
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
func run() error {
	r := mux.NewRouter()
	r.Methods("GET").Path("/journal").HandlerFunc(RequireLoggedIn(WriterHandler))
	r.Methods("POST").Path("/journal").HandlerFunc(RequireLoggedIn(SaveHandler))
	r.Path("/tie").HandlerFunc(AllTiesHandler)
	r.Path("/tie/{date}.svg").HandlerFunc(TieHandler)
	r.Path("/bwv").HandlerFunc(BWVHandler)
	r.PathPrefix("/assets/").HandlerFunc(AssetHandler)

	p := secretbookmark.New(*secret_parameter, *password_file)
	r.Use(p.Middleware)

	defer onShutdown()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
	// TODO: cleanup
	log.Printf("Shutdown complete")
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
		Contents: body,
	}

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
