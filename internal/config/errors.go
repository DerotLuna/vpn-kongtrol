package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ValidationError is a user-facing config validation error. It carries an
// internal/i18n key plus args so each client (CLI, dashboard) renders it in
// the user's language; Error() stays English for logs and existing callers.
// Every Key has a companion Key+".fix" entry: one "what to do" line. Format
// strings use explicit argument indexes (%[1]q), so a message may skip args.
type ValidationError struct {
	Key  string
	Args []any
}

func newValidationError(key string, args ...any) *ValidationError {
	return &ValidationError{Key: key, Args: args}
}

func (e *ValidationError) Error() string { return e.Message(i18n.EN) }

// Message is the short problem description in lang.
func (e *ValidationError) Message(lang i18n.Lang) string { return render(lang, e.Key, e.Args) }

// Hint is the one-line "what to do" in lang.
func (e *ValidationError) Hint(lang i18n.Lang) string {
	return render(lang, e.Key+".fix", e.Args)
}

// LocalizeError renders err in lang when its chain holds a ValidationError:
// message and hint separately. ok is false for any other error, which the
// caller keeps showing as before.
func LocalizeError(err error, lang i18n.Lang) (msg, hint string, ok bool) {
	var ve *ValidationError
	if !errors.As(err, &ve) {
		return "", "", false
	}
	return ve.Message(lang), ve.Hint(lang), true
}

// render formats key with args; a text with no verbs is returned as is, so
// a "what to do" line need not mention every arg.
func render(lang i18n.Lang, key string, args []any) string {
	s := i18n.T(lang, key)
	if !strings.Contains(s, "%") {
		return s
	}
	return fmt.Sprintf(s, args...)
}
