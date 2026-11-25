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

// An .ovpn that names ca.crt is imported with the file from its own folder
// or given after it, and a missing one is named with the CLI's way to add
// it (ADR-0105).
func TestProfileImport_ReadsReferencedFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KONGTROL_LANG", "en")
	t.Cleanup(func() { profileImportExtra = nil })
	src, other := t.TempDir(), t.TempDir()
	conf := filepath.Join(src, "office.ovpn")
	ca := "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"
	if err := os.WriteFile(conf, []byte("client\nremote vpn.example.com 1194\nca ca.crt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := runProfileImport(conf, "", false)
	if err == nil || !strings.Contains(err.Error(), "ca.crt") || !strings.Contains(err.Error(), "kongtrol profile import") {
		t.Fatalf("missing: %v", err)
	}
	elsewhere := filepath.Join(other, "ca.crt")
	if err := os.WriteFile(elsewhere, []byte(ca), 0o600); err != nil {
		t.Fatal(err)
	}
	profileImportExtra = []string{elsewhere}
	res, err := runProfileImport(conf, "a", false)
	if err != nil || len(res.Included) != 1 {
		t.Fatalf("extra file: %+v %v", res, err)
	}
	profileImportExtra = nil
	if err := os.WriteFile(filepath.Join(src, "ca.crt"), []byte(ca), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err := runProfileImport(conf, "b", false); err != nil || len(res.Included) != 1 {
		t.Fatalf("same folder: %+v %v", res, err)
	}
	// An absolute path the .ovpn names is read too (ADR-0112).
	otherReal, err := filepath.EvalSymlinks(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(src, "abs.ovpn")
	line := "ca \"" + strings.ReplaceAll(otherReal, `\`, `\\`) + "\"\n"
	if err := os.WriteFile(abs, []byte("client\nremote vpn.example.com 1194\n"+line), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err := runProfileImport(abs, "c", false); err != nil || len(res.Included) != 1 {
		t.Fatalf("absolute path: %+v %v", res, err)
	}
}
