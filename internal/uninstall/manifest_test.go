package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPurge_KeepManifestOnly(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"kongtrol.yaml", "api-token", "daemon.log", ManifestName} {
		write(t, filepath.Join(dir, f), "x: 1\n")
	}
	rep := Purge(PurgeOptions{DataDir: dir, KeepManifest: true, Delete: func(Credential) error { return nil }})
	if rep.Err() != nil {
		t.Fatal(rep.Err())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != ManifestName {
		t.Fatalf("only the manifest must stay, got %v", entries)
	}
}

func TestReadManifest_BOM(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "kongtrol.exe")
	write(t, filepath.Join(dir, ManifestName), string(rune(0xFEFF))+"file="+exe+"\r\n")
	m, ok := ReadManifest(dir)
	if !ok || len(m.Files) != 1 || m.Files[0] != exe {
		t.Fatalf("BOM/CRLF manifest not parsed: %+v", m)
	}
}

// Regression: a manifest line "file=/" or "file=$HOME" made the uninstall
// own (and delete) a parent folder. Only Kongtrol's own names are kept, and
// only Kongtrol.app may contain the executable.
func TestReadManifest_OnlyKongtrolNames(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Dir(dir)
	exe := filepath.Join(home, "bin", "kongtrol")
	app := filepath.Join(home, "Applications", "Kongtrol.app")
	body := "file=" + home + "\nfile=" + filepath.Join(home, "bin") + "\nfile=" + exe +
		"\nfile=" + app + "\nfile=" + filepath.Join(home, "notes.txt") +
		"\nautostart=" + filepath.Join(home, ".bashrc") + "\nautostart=" + filepath.Join(home, ".config", "autostart", "kongtrol.desktop") + "\n"
	write(t, filepath.Join(dir, ManifestName), body)
	m, ok := ReadManifest(dir)
	if !ok || len(m.Files) != 2 || m.Files[0] != exe || m.Files[1] != app {
		t.Fatalf("files = %v", m.Files)
	}
	if len(m.Autostart) != 1 || filepath.Base(m.Autostart[0]) != "kongtrol.desktop" {
		t.Fatalf("autostart = %v", m.Autostart)
	}
	if !manifestOwns(m, filepath.Join(app, "Contents", "MacOS", "kongtrol")) {
		t.Fatal("exe inside the listed Kongtrol.app must count")
	}
	if manifestOwns(Manifest{Files: []string{filepath.Join(home, "bin", "kongtrol-tray")}}, filepath.Join(home, "bin", "kongtrol-tray", "x")) {
		t.Fatal("a file entry is not a folder that owns the exe")
	}
}
