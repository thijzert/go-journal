go-journal
==========

This is a Journaling utility I wrote in Go. It features a command-line utility for writing, tagging, and searching a journal, as well as a web interface for taking notes on the go.

Go-journal was originally created as a replacement for [jrnl](https://github.com/maebert/jrnl), but other than the storage format the two share very little.

Overview
-----
This repository builds two executables. The first, `jrnl`, kinda sorta emulates what @maebert did. One can either pass the `--create` flag to add an entry by piping some text to stdin, or the `--search` flag to look for any journal entries that contain all of the following arguments.

The `journal-server` spins up a web server with an interface for adding entries to the journal. (Not reading them!)
It's secured via the very advanced 'secret bookmark' method, which easily enables one to use the interface on your smartphone, no matter the species. Accidental exposure of this bookmark may result in some spam entries being added, but since the web interface is write-only your journal itself remains safe from prying eyes.

There are two notable exceptions to the "write-only" policy in the web server. First, there's the special `@BWV` tag. If you're anything like me, you like keeping track of which music you've played, and in particular, which [BWV numbers](https://en.wikipedia.org/wiki/List_of_compositions_by_Johann_Sebastian_Bach#BWV) you can cross off. `journal-server` exposes a (non-exhaustive) list of BWV numbers, that turn green as they become tagged in your journal.

Second, if you specify a projects directory, the file names in that directory can be selected through a dropdown list. If a project log file is selected, the journal entry is appended to that file in addition to the journal file.

Usage
-----
### `jrnl`
Add entries to the journal, or search for past entries.

Command-line arguments:

* `--journal_file=FILE`: read or write journal entries to or from `FILE`.
* `--search`: search the journal and print matching entries. All other command-line arguments are search terms.
* `--create`: add a new journal entry. This reads input from stdin and adds it to the journal.
* `--date=DATE`: (when adding an entry) use `DATE` for the new journal entry, instead of the current date and time.

### `journal-server`
Start a web server

Command-line arguments:

* `--listen=IP:PORT`: listen on port `PORT`, on IP `IP`. Defaults to ':8848'.
* `--journal_file=FILE`: read or write journal entries to or from `FILE`. `FILE` defaults to 'journal.txt' in the current directory.
* `--password_file=FILE`: read passwords from `FILE`. This file should be in the apache htpasswd format, with bcrypt hashes. `FILE` defaults to '.htpasswd' in the current directory.
* `--secret_parameter=URLKEY`: Pass the API key in this URL parameter, making it less obvious to find and brute force. Defaults to 'apikey'
* `--attachments_dir=DIR`: Directory for storing attached files. If this parameter is not specified, attaching uploaded files is disabled.
* `--projects_dir=DIR`: Directory with project log files. If this parameter is not specified, adding entries to a project log is disabled.

Building
--------
First install or update the prerequisites:

```
go get ./...
gem install scss
```

Afterwards, execute `build.sh` to build both binaries.

`journal-server` listens on port 8848 by default, and uses the file `journal.txt` in the current directory for storage. These options can be tweaked using the `--listen` and `--journal-file` flags respectively.
Furthermore, it needs a `.htpasswd` file to verify the bookmarked API key. This file should take the Apache htpasswd format; use any standard utility to create it. (But tell it to use the bcrypt hash.)

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
