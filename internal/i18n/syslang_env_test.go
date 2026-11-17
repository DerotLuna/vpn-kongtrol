//go:build !windows

package i18n

import "testing"

func TestEnvLocalePrecedence(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "C")
	t.Setenv("LANG", "es_AR.UTF-8")
	if got := envLocale(); got != "es_AR.UTF-8" {
		t.Fatalf("neutral LC_MESSAGES must fall through to LANG, got %q", got)
	}
	t.Setenv("LC_ALL", "en_GB.UTF-8")
	if got := envLocale(); got != "en_GB.UTF-8" {
		t.Fatalf("LC_ALL must win, got %q", got)
	}
}
