package secretbookmark

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/context"
	"golang.org/x/crypto/bcrypt"
)

type SecretBookmark struct {
	parameterName string
	passwordFile  string
}

func New(parameterName, passwordFile string) *SecretBookmark {
	if parameterName == "" {
		parameterName = "apikey"
	}
	if passwordFile == "" {
		passwordFile = ".htpasswd"
	}
	return &SecretBookmark{parameterName, passwordFile}
}

func (s *SecretBookmark) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer next.ServeHTTP(w, r)

		passkey := []byte(r.URL.Query().Get(s.parameterName))
		if len(passkey) == 0 {
			return
		}
		// TODO: check passkey for validity.
		// TODO: store some sort of a username in the context

		pwds, err := os.Open(s.passwordFile)
		if err != nil {
			log.Printf("Error opening password file %s: %s", s.passwordFile, err)
			return
		}

		pwdr := bufio.NewReader(pwds)
		var line, user, phash []byte
		for ; err == nil; line, _, err = pwdr.ReadLine() {
			if len(line) < 3 || line[0] == '#' {
				continue
			}

			for i, c := range line {
				if c == ':' {
					user = line[:i]
					phash = line[i+1:]
					break
				}
			}

			if len(phash) > 5 && bytes.Equal(phash[0:4], []byte("$2y$")) {
				// Verify Bcrypt
				if err = bcrypt.CompareHashAndPassword(phash, passkey); err == nil {
					context.Set(r, "login", string(user))
					return
				}
			} else {
				log.Printf("Unknown password hash format '%s'", phash)
			}
		}
	})
}
