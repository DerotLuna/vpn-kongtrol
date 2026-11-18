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
