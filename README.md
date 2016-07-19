go-journal
==========

This is a Journaling utility I wrote in Go. It features a command-line utility for writing, tagging, and searching a journal, as well as a web interface for taking notes on the go.

Go-journal was originally created as a replacement for [jrnl](https://github.com/maebert/jrnl), but other than the storage format the two share very little.

Usage
-----
This repository builds two executables. The first, `jrnl`, kinda sorta emulates what @maebert did. One can either pass the `--create` flag to add an entry by piping some text to stdin, or the `--search` flag to look for any journal entries that contain all of the following arguments.

The `journal-server` spins up a web server with an interface for adding entries to the journal. (Not reading them!)
It's secured via the very advanced 'secret bookmark' method, which easily enables one to use the interface on your smartphone, no matter the species. Loss of this bookmark may result in some spam entries being added, but since the web interface is write-only your journal itself remains safe from prying eyes.

One notable exception to the "write-only" policy is the special `@BWV` tag. If you're anything like me, you like keeping track of which music you've played, and in particular, which [BWV numbers](https://en.wikipedia.org/wiki/List_of_compositions_by_Johann_Sebastian_Bach#BWV) you can cross off. `journal-server` exposes a (non-exhaustive) list of BWV numbers, that turn green as they become tagged in your journal.

Building
--------
First install or update the prerequisites:

```
gem install scss
go get -u github.com/jteeuwen/go-bindata/...
go get -u github.com/gorilla/mux
go get -u github.com/gorilla/context
go get -u golang.org/x/crypto/bcrypt
```

Afterwards, execute `build.sh` to build both binaries.

`journal-server` listens on port 8848 by default, and uses the file `journal.txt` in the current directory for storage. These options can be tweaked using the `--listen` and `--journal-file` flags respectively.
Furthermore, it needs a `.htpasswd` file to verify the bookmarked API key. Use your favorite utility to create it. (Use the bcrypt hash!)

If you just want to quickly give it a go, create a file `.htpasswd` with the following contents:

```
random-github-user:$2y$11$nEGADAX66RFYfTJd1b7LZuNM3zD9PxAJRnxFnEQ3vsDc3s9u7jMfm
```

run the binary, and point your browser to: http://localhost:8848/journal?apikey=lalala .

