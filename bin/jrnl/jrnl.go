package main

import (
	"flag"
	"github.com/thijzert/go-journal"
	"io/ioutil"
	"os"
	"strings"
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
		// Remove trailing newlines from the contents
		for len(c) > 0 && c[len(c)-1] == 0x0a {
			c = c[0 : len(c)-1]
		}
		// Remove carriage returns entirely. Why? Because it fits my use case, and because sod MS-DOS.
		conts := strings.Replace(string(c), "\r", "", -1)

		e := &journal.Entry{
			Date:     t,
			Starred:  false,
			Contents: conts}

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
