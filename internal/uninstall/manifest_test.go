package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPurge_KeepManifestOnly(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"kongtrol.yaml", "api-token", "daemon.log", "webview2/EBWebView/Default/Cache/journal.baj", ManifestName} {
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

// install.ps1 with the tray (ADR-0075): tray exe, Start menu shortcut and
// the Run value are listed; a runvalue= that is not a kongtrol-tray.exe and
// keys only the scripts read (version=, components=) are ignored.
func TestReadManifest_WindowsTray(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	exe, tray := filepath.Join(bin, "kongtrol.exe"), filepath.Join(bin, "kongtrol-tray.exe")
	lnk := filepath.Join(dir, "Programs", "Kongtrol.lnk")
	body := "file=" + exe + "\nfile=" + tray + "\nfile=" + lnk + "\nfile=" + filepath.Join(dir, "Programs", "Other.lnk") +
		"\nrunvalue=" + tray + "\nrunvalue=" + filepath.Join(bin, "evil.exe") + "\nrunvalue=relative\\kongtrol-tray.exe" +
		"\ncomponents=cli,tray\nversion=0.8.0\npath=" + bin + "\n"
	write(t, filepath.Join(dir, ManifestName), body)
	m, ok := ReadManifest(dir)
	if !ok || len(m.Files) != 3 || m.Files[2] != lnk {
		t.Fatalf("files = %v", m.Files)
	}
	if len(m.RunValues) != 1 || m.RunValues[0] != tray {
		t.Fatalf("run values = %v", m.RunValues)
	}
	p := Detect(Env{GOOS: "windows", Exe: exe, DataDir: dir, Exists: func(string) bool { return false }}, false, false)
	if p.Channel != ChannelScript || len(p.RunValues) != 1 || len(p.PathEntries) != 1 {
		t.Fatalf("plan = %+v", p)
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
