package main

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/thijzert/go-journal"
	"github.com/thijzert/go-journal/bach"
)

type bwvCheck struct {
	BWV  bach.Comp
	Done bool
}

func BWVHandler(w http.ResponseWriter, r *http.Request) {
	doneDeal := make(map[string]bool)
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

			doneDeal[norm] = true
		}
	}

	bwvData := struct {
		BWVs [][]bwvCheck
	}{make([][]bwvCheck, 0)}

	for _, sect := range bach.AllCantatas {
		ns := make([]bwvCheck, len(sect))
		for i, c := range sect {
			ns[i] = bwvCheck{c, doneDeal[c.BWV]}
		}
		bwvData.BWVs = append(bwvData.BWVs, ns)
	}

	executeTemplate(bwvlist, bwvData, w, r)
}
