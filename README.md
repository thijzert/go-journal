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

### Custom launcher script
If you're anything like me, you lie awake at night worrying about certificate forgeries.
Also, while the web interface may look okay on your phone, its design may not be all that convenient on bigger screens. One could argue that some sort of responsiveness in the style is called for, but you could also create a script that launches a custom browser window and bind it to a hotkey. For instance:

```bash
#!/bin/bash

TD=$(mktemp -d)

/usr/bin/chromium-browser \
	--incognito --disable-java --disable-client-side-phishing-detection --disable-translate \
	--no-first-run --disable-restore-session-state --no-default-browser-check \
	--window-size="335,450" \
	--user-data-dir="$TD" \
	--app="https://your.server/journal?apikey=YouKnowILearnedSomethingToday" \
	--ssl-version-min="tls1.2" \
	--hsts-hosts='{
		"/HSD/qf3UlD33yF7IbYl+Tc4w8Fu+mNUYL8m7yTxWKg=": {
			"dynamic_spki_hashes": [ "sha256/+Luw7m6APH/JszU+Yqj7zyGbCok0ZYMaKhkYmNtrKHI=", "sha256/dd+XFi9YBvUS5vB1yfv4DpSRnbPBuXenp6LPkaKXFlg=" ],
			"dynamic_spki_hashes_expiry": 1450884384.287938,
			"expiry": 1450884384.287909,
			"mode": "force-https",
			"pkp_include_subdomains": false,
			"pkp_observed": 1419348384.287938,
			"sts_include_subdomains": true,
			"sts_observed": 1419348384.287909
		}
	}'

rm -rf "$TD"
```
