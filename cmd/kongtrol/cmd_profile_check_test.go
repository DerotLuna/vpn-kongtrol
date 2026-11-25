package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// kongtrol profile import --check runs the same import and saves nothing
// (ADR-0118): no folder, no file; the same translated refusals; a taken
// name is reported, not refused.
func TestProfileImportCheck_SavesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KONGTROL_LANG", "es")
	profileImportCheck = true
	t.Cleanup(func() { profileImportCheck = false })
	src := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(src, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	conf := write("oficina.ovpn", "client\nremote vpn.example.com 1194\nca ca.crt\nauth-user-pass\n")
	write("ca.crt", "-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n")

	res, err := runProfileImport(conf, "", false)
	if err != nil || !res.Check || res.Path != "" || res.File != "oficina.ovpn" || !res.NeedsCredentials || len(res.Included) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	configs := filepath.Join(home, ".kongtrol", "configs")
	if _, err := os.Stat(configs); !os.IsNotExist(err) {
		t.Fatalf("--check created %s: %v", configs, err)
	}
	hook := write("malo.conf", "[Interface]\nPrivateKey = a\nPostUp = calc.exe\n")
	if _, err := runProfileImport(hook, "", false); err == nil || !strings.Contains(err.Error(), "PostUp") {
		t.Fatalf("hook: %v", err)
	}

	profileImportCheck = false
	if _, err := runProfileImport(conf, "", false); err != nil {
		t.Fatal(err)
	}
	profileImportCheck = true
	if res, err := runProfileImport(conf, "", false); err != nil || !res.Exists {
		t.Fatalf("taken name: %+v %v", res, err)
	}
}
