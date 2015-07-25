package main

import (
	"flag"
	"github.com/ametheus/go-journal"
	"io/ioutil"
	"os"
)

var (
	journal_file = flag.String("journal_file", "journal.txt", "Journal File")
	act_create   = flag.Bool("create", false, "Create a new entry")
	act_search   = flag.Bool("search", false, "Search the journal for entries matching these tags")
	date         = flag.String("date", "", "Date/time of new entry")
)

func main() {
	flag.Parse()

	if !*act_create && !*act_search {
		panic("Specify at least one action (--create, --search, etc)")
	}
	if *act_create && *act_search {
		panic("Can't search and create a new entry.")
	}

	if *act_create {
		t := journal.SmartTime(*date)
		c, _ := ioutil.ReadAll(os.Stdin)

		e := &journal.Entry{
			Date:     t,
			Starred:  false,
			Contents: string(c)}

		err := journal.Add(*journal_file, e)
		if err != nil {
			panic(err)
		}
	}
	if *act_search {
		terms := flag.Args()

		result, err := journal.Search(*journal_file, terms...)
		if err != nil {
			panic(err)
		}
		var i int = 0

		for e := range result {
			// TODO: nicer formatting
			// TODO: detect a pipe, and fall back to non-nice formatting.
			if i > 0 {
				os.Stdout.Write([]byte("\n"))
			}
			e.Serialize(os.Stdout)
			i++
		}

		if i == 0 {
			os.Exit(1)
		}
	}
}
