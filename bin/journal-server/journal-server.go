package main

import (
	"flag"
	"fmt"
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
	r.Path("/tie").HandlerFunc(AllTiesHandler)
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

func AllTiesHandler(w http.ResponseWriter, r *http.Request) {
	y := time.Now().Year()
	ymax := y + 4
	w.Header()["Content-Type"] = []string{"text/html"}
	w.WriteHeader(200)
	fmt.Fprintf(w, "<html><div>")
	var m time.Month = 0
	dt, _ := time.ParseInLocation("2006-01-02", fmt.Sprintf("%d-01-01", y), time.UTC)
	y = 0
	for dt.Year() < ymax {
		if dt.Year() != y {
			y = dt.Year()
			fmt.Fprintf(w, "</div></div><div style=\"float: left; width: 224px; margin: 0 20px;\"><h1>%d</h1><div>", y)
		}
		if dt.Month() != m {
			fmt.Fprintf(w, "</div><h4>%s</h4><div>", dt.Format("January"))
			wd := (int(dt.Weekday()) + 6) % 7
			for i := 0; i < wd; i++ {
				fmt.Fprintf(w, "<div style=\"display: inline-block; width: 32px; height: 32px\"></div>")
			}

			m = dt.Month()
		}
		if dt.Weekday() == time.Monday {
			fmt.Fprintf(w, "</div><div>")
		}
		dfe := daysSinceEaster(dt)
		fmt.Fprintf(w, "<img src=\"tie/%s.svg\" style=\"width: 32px; height: 32px\" data-days-since-easter=\"%d\"/>", dt.Format("2006-01-02"), dfe)
		dt = dt.AddDate(0, 0, 1)
	}
	fmt.Fprintf(w, "</div></div></html>")
}

func eastersunday(year int) (month, day int) {
	a := year % 19
	b := year % 4
	c := year % 7
	k := year / 100
	p := (13 + 8*k) / 25
	q := k / 4
	M := (15 - p + k - q) % 30
	N := (4 + k - q) % 7
	d := (19*a + M) % 30
	e := (2*b + 4*c + 6*d + N) % 7

	month = 3
	day = 22 + d + e
	if day > 31 {
		day -= 31
		month++
	}
	return
}

func daysSinceEaster(dt time.Time) int {
	m, d := eastersunday(dt.Year())
	ea := time.Date(dt.Year(), time.Month(m), d, 0, 0, 0, 0, time.UTC)
	diff := dt.Sub(ea)
	return int(int64(diff) / int64(24*time.Hour))
}

const (
	GOLD   string = "gold"
	GREEN         = "teal"
	RED           = "red"
	PURPLE        = "purple"
	BLUE          = "blue"
	WHITE         = "#E8EAF3"
	BLACK         = "#151B14"
)

func tieColour(dt time.Time) string {
	y, m, d := dt.Date()
	wd := dt.Weekday()

	if m == time.January && d <= 6 {
		return GOLD // days 7-12 of Christmas
	}
	if m == time.January && d < 14 && wd == time.Sunday {
		return WHITE // Baptism of Jesus Sunday
	}

	ea := daysSinceEaster(dt)
	if ea == 0 || ea == 1 {
		return WHITE // Easter Sunday
	} else if ea < 0 {
		if ea == -49 {
			return WHITE // Transfiguration Sunday
		} else if ea == -46 {
			return BLACK // Ash Wednesday
		} else if ea == -7 || ea == -3 || ea == -2 {
			return RED // Palm Sunday / Maundy Thursday / Good Friday
		} else if ea > -46 {
			return PURPLE
		}
	} else if ea == 39 {
		return RED // Heaventravelday
	} else if ea < 49 {
		return GOLD // Season of Easter
	} else if ea == 49 || ea == 50 {
		return RED // Pentecost
	} else if ea == 56 {
		return WHITE // Trinity Sunday
	}

	if m == time.November && d == 1 {
		return WHITE // All Saints
	}

	reignOfChrist := time.Date(y, time.November, 30, 0, 0, 0, 0, time.UTC)
	offset := 7 - int(reignOfChrist.Weekday())
	if offset > 3 {
		offset -= 7
	}
	reignOfChrist = reignOfChrist.AddDate(0, 0, offset)

	if m == reignOfChrist.Month() && d == reignOfChrist.Day() {
		return WHITE // Reign of Christ Sunday
	} else if m == time.December && d >= 25 {
		return GOLD // Days 1-6 of Christmas
	} else if dt.After(reignOfChrist) {
		return BLUE // Advent
	}

	return GREEN
}

func TieHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var date time.Time

	if vars["date"] == "today" {
		date = time.Now()
	} else if vars["date"] == "tomorrow" {
		date = time.Now().AddDate(0, 0, 1)
	} else if vars["date"] == "dayaftertomorrow" {
		date = time.Now().AddDate(0, 0, 2)
	} else if vars["date"] == "yesterday" {
		date = time.Now().AddDate(0, 0, -1)
	} else {
		var err error
		date, err = time.Parse("2006-01-02", vars["date"])

		if err != nil {
			w.Header()["Content-Type"] = []string{"text/plain"}
			w.WriteHeader(404)
			w.Write([]byte("No tie was found for that day.\n\nLive a little; wear a t-shirt.\n"))
		}
	}

	y, m, d := date.Date()
	date = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	w.Header()["Content-Type"] = []string{"image/svg+xml"}

	tieData := struct {
		Colour string
	}{tieColour(date)}

	tie.Execute(w, tieData)
}
