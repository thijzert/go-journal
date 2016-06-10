package main

import (
	"flag"
	"github.com/gorilla/context"
	"html/template"
	"log"
	"net/http"
)

var editor *template.Template

func init() {
	flag.Parse()

	b, err := Asset("assets/templates/editor.html")
	if err != nil {
		log.Fatal(err)
	}

	funcs := template.FuncMap{}

	editor, err = template.New("editor").Funcs(funcs).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}
}

func RequireLoggedIn(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		login := context.Get(r, "login")
		log.Printf("Checking login for %s; logged in: %v", r.URL.String(), login)
		if login == nil {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Access denied."))
		} else {
			f(w, r)
		}
	}
}

func executeTemplate(tpl *template.Template, data interface{}, w http.ResponseWriter, r *http.Request) {
	w.Header()["Content-Type"] = []string{"text/html; charset=UTF-8"}

	err := tpl.Execute(w, data)
	if err != nil {
		errorHandler(err, w, r)
		return
	}
}

func AssetHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	b, err := Asset(path)
	if err != nil {
		errorHandler(err, w, r)
		return
	}

	lp := len(path)
	// TODO: Isn't there middleware that handles this?
	if lp > 4 && path[lp-4:] == ".css" {
		w.Header()["Content-Type"] = []string{"text/css"}
	} else if lp > 3 && path[lp-3:] == ".js" {
		w.Header()["Content-Type"] = []string{"application/javascript"}
	} else if lp > 4 && (path[lp-4:] == ".png" || path[lp-4:] == ".jpg" || path[lp-4:] == ".svg") {
		w.Header()["Content-Type"] = []string{"image/" + path[lp-3:]}
	} else {
		w.Header()["Content-Type"] = []string{"application/octet-stream"}
	}

	w.Write(b)
}

func errorHandler(e error, w http.ResponseWriter, r *http.Request) {
	log.Print(e)
	w.WriteHeader(400)
	w.Write([]byte("TODO: error handling\n"))
	w.Write([]byte(e.Error()))
}
