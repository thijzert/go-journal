package secretbookmark

import (
	"github.com/gorilla/context"
	"net/http"
)

type SecretBookmarkHandler struct {
	handler       http.Handler
	parameterName string
	passwordFile  string
}

func NewHandler(handler http.Handler, parameterName, passwordFile string) *SecretBookmarkHandler {
	if parameterName == "" {
		parameterName = "apikey"
	}
	if passwordFile == "" {
		passwordFile = ".htpasswd"
	}
	return &SecretBookmarkHandler{handler, parameterName, passwordFile}
}

func (s *SecretBookmarkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	passkey := r.URL.Query().Get(s.parameterName)
	if passkey != "" {
		// TODO: check passkey for validity.
		// TODO: store some sort of a username in the context

		context.Set(r, "login", "yes.")
	}
	s.handler.ServeHTTP(w, r)
}
