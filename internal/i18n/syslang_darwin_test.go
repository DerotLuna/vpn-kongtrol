//go:build darwin

package i18n

import "testing"

func TestFirstAppleLanguage(t *testing.T) {
	out := "(\n    \"es-MX\",\n    \"en-US\"\n)\n"
	if got := firstAppleLanguage(out); got != "es-MX" {
		t.Fatalf("got %q", got)
	}
	if got := firstAppleLanguage(""); got != "" {
		t.Fatalf("empty output, got %q", got)
	}
}
