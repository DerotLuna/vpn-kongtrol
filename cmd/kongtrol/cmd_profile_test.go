package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// kongtrol profile import stores the file where the dashboard's import
// does (~/.kongtrol/configs), refuses what the API refuses (translated,
// with what to do) and never replaces a file without --overwrite.
func TestProfileImport_SameRulesAsTheDashboard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KONGTROL_LANG", "es")
	src := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(src, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := write("oficina.conf", "[Interface]\nPrivateKey = a\nAddress = 10.0.0.2/32\n")

	res, err := runProfileImport(good, "", false)
	want := filepath.Join(home, ".kongtrol", "configs", "oficina.conf")
	if err != nil || res.Path != want || res.Type != "wireguard" {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := runProfileImport(good, "", false); err == nil || !strings.Contains(err.Error(), "oficina.conf") || !strings.Contains(err.Error(), "\n    ") {
		t.Fatalf("second import without --overwrite: %v", err)
	}
	if res, err := runProfileImport(good, "", true); err != nil || !res.Overwritten {
		t.Fatalf("overwrite: %+v %v", res, err)
	}
	if res, err := runProfileImport(good, "casa", false); err != nil || filepath.Base(res.Path) != "casa.conf" {
		t.Fatalf("--name: %+v %v", res, err)
	}

	hook := write("x.conf", "[Interface]\nPrivateKey = a\nPostUp = calc.exe\n")
	if _, err := runProfileImport(hook, "", false); err == nil || !strings.Contains(err.Error(), "PostUp") || !strings.Contains(err.Error(), "seguridad") {
		t.Fatalf("hook: %v", err)
	}
	if _, err := runProfileImport(write("notes.txt", "x"), "", false); err == nil || !strings.Contains(err.Error(), ".ovpn") {
		t.Fatalf("extension: %v", err)
	}
	if _, err := runProfileImport(filepath.Join(src, "missing.ovpn"), "", false); err == nil {
		t.Fatal("missing file accepted")
	}
}
