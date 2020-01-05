package main

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/thijzert/go-journal"
	"github.com/thijzert/go-journal/bach"
)

type concert struct {
	Date        string
	Description string
}

type bwvCheck struct {
	BWV      bach.Comp
	Done     bool
	Concerts []concert
}

func BWVHandler(w http.ResponseWriter, r *http.Request) {
	doneDeal := make(map[string][]concert)
	rbwv := regexp.MustCompile("@BWV\\s+((([Aa]nh\\.?)\\s*)?(\\d+)([a-zA-Z])?(-\\d+)?)")

	c := make(chan *journal.Entry, 20)
	f, err := os.Open(*journal_file)
	defer f.Close()
	if err != nil {
		errorHandler(err, w, r)
		return
	}
	go func() {
		journal.Deserialize(f, c)
	}()

	for e := range c {
		mm := rbwv.FindAllStringSubmatch(e.Contents, -1)
		for _, m := range mm {
			// Normalize the BWV notation
			norm := m[4] + strings.ToLower(m[5]) + m[6]
			if m[3] != "" {
				norm = "Anh. " + norm
			}

			conc := concert{
				Date: e.Date.Format("2006-01-02"),
			}

			if time.Since(e.Date) > 1*365*24*time.Hour {
				lines := strings.Split(e.Contents, "\n")
				for _, l := range lines {
					if len(l) < 5 || l[0:4] == "@BWV" {
						continue
					}
					conc.Description = l
					break
				}
			} else if time.Since(e.Date) < 14*24*time.Hour {
				conc.Date = "xxxx-xx-xx"
			}

			doneDeal[norm] = append(doneDeal[norm], conc)
		}
	}

	bwvData := struct {
		BWVs [][]bwvCheck
	}{make([][]bwvCheck, 0)}

	for _, sect := range bach.AllCantatas {
		ns := make([]bwvCheck, len(sect))
		for i, c := range sect {
			dd, ok := doneDeal[c.BWV]
			ns[i] = bwvCheck{
				BWV:      c,
				Done:     ok && len(dd) > 0,
				Concerts: dd,
			}
		}
		bwvData.BWVs = append(bwvData.BWVs, ns)
	}

	executeTemplate(bwvlist, bwvData, w, r)
}
