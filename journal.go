package journal

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
)

const (
	dateFormat = "2006-01-02 15:04"
)

type Entry struct {
	Date     time.Time
	Starred  bool
	Contents string
}

func SmartTime(t string) time.Time {
	// TODO: Smart time
	// Parse things like 'Yesterday 15:16' or 'Thursday 2PM'

	tt, err := time.ParseInLocation(dateFormat, t, time.Local)
	if err == nil && tt.Year() > 1980 {
		return tt
	}

	return time.Now()
}

func (e *Entry) Serialize(w io.Writer) error {
	_, er := w.Write([]byte(e.Date.Format(dateFormat)))
	if er != nil {
		return er
	}

	if e.Starred {
		_, er = w.Write([]byte(" * "))
	} else {
		_, er = w.Write([]byte(" "))
	}
	if er != nil {
		return er
	}

	_, er = w.Write([]byte(e.Contents))
	if er != nil {
		return er
	}

	return nil
}

func Deserialize(r io.Reader, c chan *Entry) error {
	rr := bufio.NewReader(r)

	var err error = nil
	var line string
	var ent *Entry = nil

	var emptyLines int = 1
	var t time.Time

	for err == nil {
		line, err = rr.ReadString('\n')
		if err != nil {
			break
		}

		if line == "\n" {
			emptyLines++
			continue
		}

		if emptyLines > 0 && len(line) > len(dateFormat) {
			t, err = time.ParseInLocation(dateFormat, line[0:len(dateFormat)], time.Local)

			// Datum na een lege regel -> nieuw bericht
			if err == nil && t.Year() > 1980 {
				if ent != nil {
					c <- ent
				}

				ent = &Entry{Date: t}
				if line[len(dateFormat):len(dateFormat)+3] == " * " {
					ent.Starred = true
					ent.Contents = line[len(dateFormat)+3:]
				} else {
					ent.Contents = line[len(dateFormat)+1:]
				}

				continue
			}
		}

		if emptyLines > 0 {
			ent.Contents += "\n"
		}
		ent.Contents += line
	}

	if ent != nil {
		c <- ent
	}

	close(c)

	if err != io.EOF {
		return err
	}

	return nil
}

func Search(filename string, terms ...string) (chan *Entry, error) {
	c := make(chan *Entry, 20)
	rv := make(chan *Entry, 20)

	go func() {
		f, _ := os.Open(filename)
		defer f.Close()

		Deserialize(f, c)
	}()
	go func() {
	Found:
		for ee := range c {
			for _, t := range terms {
				if strings.Index(ee.Contents, t) == -1 {
					continue Found
				}
			}
			rv <- ee
		}

		close(rv)
	}()

	return rv, nil
}

func Add(filename string, entry *Entry) error {
	var f *os.File
	var err error

	fi, _ := os.Stat(filename)
	if fi == nil {
		f, _ = os.Open(os.DevNull)
	} else {
		f, err = os.Open(filename)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	g, err := os.Create(filename + "~")
	defer g.Close()
	if err != nil {
		return err
	}

	c := make(chan *Entry, 25)
	go Deserialize(f, c)

	for ee := range c {
		if entry != nil && ee.Date.After(entry.Date) {
			err = entry.Serialize(g)
			g.Write([]byte{0x0a})
			entry = nil
			if err != nil {
				return err
			}
		}

		err = ee.Serialize(g)
		g.Write([]byte{0x0a})
		if err != nil {
			return err
		}
	}

	if entry != nil {
		err = entry.Serialize(g)
		if err != nil {
			return err
		}
	}

	// Kopieer het nieuwe bestand naar het oude.
	f, err = os.Open(filename + "~")
	g, err = os.Create(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(g, f)
	if err != nil {
		return err
	}

	return nil
}
