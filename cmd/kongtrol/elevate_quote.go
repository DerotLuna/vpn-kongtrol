package main

import (
	"errors"
	"strings"
)

// shellQuote quotes s as one /bin/sh word, wrapping it in single quotes
// and writing each embedded single quote as quote, backslash-quote, quote.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// appleScriptString renders s as an AppleScript string literal.
func appleScriptString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// appleScriptAdminCommand builds the osascript source that runs argv as
// root behind the macOS administrator dialog. Every argument is
// shell-quoted, then the whole command is escaped for the AppleScript
// literal. Control characters are refused rather than quoted.
func appleScriptAdminCommand(argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("empty command")
	}
	words := make([]string, len(argv))
	for i, a := range argv {
		if strings.ContainsAny(a, "\x00\r\n") {
			return "", errors.New("argument contains a control character")
		}
		words[i] = shellQuote(a)
	}
	return "do shell script " + appleScriptString(strings.Join(words, " ")) +
		" with administrator privileges", nil
}
