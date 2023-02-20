package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/context"
)

var index *template.Template
var editor *template.Template
var daily *template.Template
var bwvlist *template.Template
var tie *template.Template

func stripProjectSuffix(name string) string {
	if len(name) > 4 && name[len(name)-4:] == ".txt" {
		name = name[:len(name)-5]
	} else if len(name) > 5 && name[len(name)-5:] == ".wiki" {
		name = name[:len(name)-5]
	}
	return name
}

func formatProjectName(name string) string {
	name = stripProjectSuffix(name)
	name = strings.Replace(name, "_", " ", -1)

	return name
}

func init() {
	flag.Parse()

	funcs := template.FuncMap{}
	funcs["ProjectName"] = formatProjectName

	b, err := Asset("assets/templates/editor.html")
	if err != nil {
		log.Fatal(err)
	}
	editor, err = template.New("editor").Funcs(funcs).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

	b, err = Asset("assets/templates/daily.html")
	if err != nil {
		log.Fatal(err)
	}
	daily, err = template.New("daily").Funcs(funcs).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

	b, err = Asset("assets/templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	index, err = template.New("index").Funcs(funcs).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

	b, err = Asset("assets/templates/bwvlist.html")
	if err != nil {
		log.Fatal(err)
	}
	bwvlist, err = template.New("bwvlist").Funcs(funcs).Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}

	b, err = Asset("assets/templates/tie.svg")
	if err != nil {
		log.Fatal(err)
	}
	tie, err = template.New("tie").Parse(string(b))
	if err != nil {
		log.Fatal(err)
	}
}

func RequireLoggedIn(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		login := context.Get(r, "login")
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

func writeJSON(w http.ResponseWriter, val any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(val)
}

func writeJSONError(w http.ResponseWriter, statusCode int, errorCode int, message string) {
	val := struct {
		Error   int    `json:"error"`
		Message string `json:"_"`
		OK      int    `json:"ok"`
	}{errorCode, message, 0}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	enc.Encode(val)
}
