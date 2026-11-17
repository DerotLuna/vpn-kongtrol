package daemonctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestFindKongtrol_FallsBackToPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, executableName)
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := FindKongtrol(i18n.EN)
	if err != nil {
		t.Fatalf("FindKongtrol: %v", err)
	}
	if !strings.EqualFold(filepath.Clean(got), filepath.Clean(exe)) {
		t.Fatalf("got %s, want %s", got, exe)
	}
}

func TestFindKongtrol_NotFoundIsTranslated(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := FindKongtrol(i18n.ES)
	if err == nil || !strings.Contains(err.Error(), "PATH") || !strings.Contains(err.Error(), "no se encontró") {
		t.Fatalf("err=%v, want the Spanish not-found message", err)
	}
}

func TestAPIBase_WildcardDialsLoopback(t *testing.T) {
	if got := APIBase("0.0.0.0", 9741); got != "http://127.0.0.1:9741" {
		t.Fatalf("got %s", got)
	}
	if got := APIBase("::1", 9741); got != "http://[::1]:9741" {
		t.Fatalf("got %s", got)
	}
}
