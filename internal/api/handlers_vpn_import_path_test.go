package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func pathBody(path string, elevate bool) string {
	b, _ := json.Marshal(importReq{Path: path, Elevate: elevate})
	return string(b)
}

// A root-only file (/etc/wireguard/wg0.conf, #5 #7) is refused with the
// way out until the user accepts the permission window ("elevate"); the
// elevated content goes through the same checks (ADR-0058).
func TestImportVPNFile_PathAndRootOnly(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := filepath.Join(t.TempDir(), "configs")
	s.SetImportDir(dir)
	src := filepath.Join(t.TempDir(), "wg0.conf")
	if err := os.WriteFile(src, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}

	// Readable: imported under the file's name.
	if code, out := doImport(t, s, pathBody(src, false)); code != http.StatusCreated || out["path"] != filepath.Join(dir, "wg0.conf") {
		t.Fatalf("readable path: %d %v", code, out)
	}
	if code, out := doImport(t, s, pathBody("wg0.conf", false)); code != http.StatusBadRequest || out["error_key"] != "api.err.import_protected_path" {
		t.Fatalf("relative path: %d %v", code, out)
	}

	orig := importReadFile
	t.Cleanup(func() { importReadFile = orig })
	importReadFile = func(string) ([]byte, error) { return nil, &fs.PathError{Op: "open", Err: fs.ErrPermission} }
	body := func(elevate bool) string {
		b, _ := json.Marshal(importReq{Path: src, Elevate: elevate, Name: "office", Overwrite: true})
		return string(b)
	}
	if code, out := doImport(t, s, body(false)); code != http.StatusForbidden || out["error_key"] != "config.err.file_root_only" || out["hint"] == "" {
		t.Fatalf("root-only without elevate: %d %v", code, out)
	}
	// No reader on this OS: same answer, never a silent read.
	if code, out := doImport(t, s, body(true)); code != http.StatusForbidden || out["error_key"] != "config.err.file_root_only" {
		t.Fatalf("elevate without reader: %d %v", code, out)
	}
	s.SetProtectedReader(func(context.Context, string) ([]byte, error) { return nil, errors.New("pkexec: dismissed") })
	if code, out := doImport(t, s, body(true)); code != http.StatusForbidden || out["error_key"] != "api.err.import_elevated_failed" {
		t.Fatalf("cancelled: %d %v", code, out)
	}
	s.SetProtectedReader(func(context.Context, string) ([]byte, error) {
		return []byte("[Interface]\nPrivateKey = a\nPostUp = id\n"), nil
	})
	if code, out := doImport(t, s, body(true)); code != http.StatusBadRequest || out["error_key"] != "api.err.import_hook" {
		t.Fatalf("elevated content is checked again: %d %v", code, out)
	}
	s.SetProtectedReader(func(context.Context, string) ([]byte, error) { return []byte(wgConf), nil })
	if code, out := doImport(t, s, body(true)); code != http.StatusCreated || out["path"] != filepath.Join(dir, "office.conf") {
		t.Fatalf("elevated: %d %v", code, out)
	}
}
