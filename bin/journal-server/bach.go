package main

import (
	"github.com/thijzert/go-journal/bach"
	"net/http"
)

type bwvCheck struct {
	BWV  bach.Comp
	Done bool
}

func BWVHandler(w http.ResponseWriter, r *http.Request) {
	bwvData := struct {
		BWVs [][]bwvCheck
	}{make([][]bwvCheck, 0)}

	for _, sect := range bach.AllCantatas {
		ns := make([]bwvCheck, len(sect))
		for i, c := range sect {
			// TODO: find out if we've done this one
			ns[i] = bwvCheck{c, false}
		}
		bwvData.BWVs = append(bwvData.BWVs, ns)
	}

	executeTemplate(bwvlist, bwvData, w, r)
}
