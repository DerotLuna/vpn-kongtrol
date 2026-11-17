package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// requestLang is the language the dashboard asked for: the X-Kongtrol-Lang
// header (set by the page from its own language toggle), else the browser's
// Accept-Language. Anything not Spanish renders in English.
func requestLang(r *http.Request) i18n.Lang {
	h := r.Header.Get("X-Kongtrol-Lang")
	if h == "" {
		h = r.Header.Get("Accept-Language")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(h)), "es") {
		return i18n.ES
	}
	return i18n.EN
}

// writeLocalizedError writes err like writeError, but a
// config.ValidationError is rendered in the request's language with its
// i18n key and a one-line "what to do" hint:
// {"error": msg, "hint": hint, "error_key": key}.
func writeLocalizedError(w http.ResponseWriter, r *http.Request, status int, err error) {
	var ve *config.ValidationError
	if !errors.As(err, &ve) {
		writeError(w, status, err.Error())
		return
	}
	lang := requestLang(r)
	writeJSON(w, status, map[string]string{
		"error":     ve.Message(lang),
		"hint":      ve.Hint(lang),
		"error_key": ve.Key,
	})
}

// writeGuidedError writes err like writeError, but an error with a
// privops guidance key (helper missing, tunnel not Kongtrol's, config file
// not yours, …) becomes one translated line in the request's language:
// {"error": line, "error_key": key, "detail": raw English error}.
func writeGuidedError(w http.ResponseWriter, r *http.Request, status int, err error) {
	key, ok := privops.GuidanceKey(err)
	if !ok {
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, status, map[string]string{
		"error":     i18n.T(requestLang(r), key),
		"error_key": key,
		"detail":    err.Error(),
	})
}
