package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkBody(name, typ, content string) string {
	b, _ := json.Marshal(importReq{Name: name, Type: typ, ContentBase64: base64.StdEncoding.EncodeToString([]byte(content)), Check: true})
	return string(b)
}

// "check": true runs the import and writes nothing (ADR-0118): 200 with
// what the import would do; the same refusals with the same bodies; a taken
// name is a field, not a 409.
func TestImportCheck_WritesNothing(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := filepath.Join(t.TempDir(), "configs")
	s.SetImportDir(dir)

	code, out := doImport(t, s, checkBody("office", "openvpn", ovpnConf+"auth-user-pass\n"))
	if code != http.StatusOK || out["status"] != "checked" || out["type"] != "openvpn" || out["file"] != "office.ovpn" ||
		out["exists"] != false || out["needs_credentials"] != true || out["path"] != nil {
		t.Fatalf("%d %v", code, out)
	}
	if inc, ok := out["included"].([]any); !ok || len(inc) != 0 {
		t.Fatalf("included = %v", out["included"])
	}
	if miss, ok := out["missing"].([]any); !ok || len(miss) != 0 {
		t.Fatalf("missing = %v", out["missing"])
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("a check created the folder: %v", err)
	}

	// Same refusal as the import, with "missing".
	code, out = doImport(t, s, checkBody("x", "openvpn", "client\nca /kongtrol-missing/ca.crt\n"))
	if code != http.StatusBadRequest || out["error_key"] != "api.err.import_missing_files" || out["hint"] == "" {
		t.Fatalf("%d %v", code, out)
	}
	if miss, _ := out["missing"].([]any); len(miss) != 1 {
		t.Fatalf("missing = %v", out["missing"])
	}
	if code, out = doImport(t, s, checkBody("x", "wireguard", "[Interface]\nPrivateKey = a\nPostUp = calc.exe\n")); code != http.StatusBadRequest || out["error_key"] != "api.err.import_hook" {
		t.Fatalf("%d %v", code, out)
	}

	// A stored file of that name: reported as "exists", nothing replaced.
	if code, _ = doImport(t, s, importBody("home", "wireguard", wgConf, false)); code != http.StatusCreated {
		t.Fatal(code)
	}
	if code, out = doImport(t, s, checkBody("home", "wireguard", wgConf+"# v2\n")); code != http.StatusOK || out["exists"] != true {
		t.Fatalf("%d %v", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "home.conf")); string(b) != wgConf {
		t.Fatalf("a check replaced the file: %q", b)
	}
}

// A check never opens the OS permission window: a root-only file answers
// config.err.file_root_only even with "elevate".
func TestImportCheck_NeverElevates(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	src := filepath.Join(t.TempDir(), "wg0.conf")
	if err := os.WriteFile(src, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := importReadFile
	t.Cleanup(func() { importReadFile = orig })
	importReadFile = func(string) ([]byte, error) { return nil, &fs.PathError{Op: "open", Err: fs.ErrPermission} }
	s.SetProtectedReader(func(context.Context, string) ([]byte, error) {
		t.Error("a check asked for permission")
		return nil, fs.ErrPermission
	})
	b, _ := json.Marshal(importReq{Path: src, Elevate: true, Check: true})
	if code, out := doImport(t, s, string(b)); code != http.StatusForbidden || out["error_key"] != "config.err.file_root_only" {
		t.Fatalf("%d %v", code, out)
	}
}

// A check is audited as config.profile.import_check (INFO, never the body)
// and announces no change; a real import stays config.profile.import.
func TestImportCheck_AuditedAsCheckNoChange(t *testing.T) {
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "test-control-token"}
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	sink := &auditSink{}
	s.SetAuditLog(sink.add)
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+importPath, s.handleImportVPNFile)
	h := s.secureHandler(mux)

	if code := send(h, http.MethodPost, importPath, s.apiToken, checkBody("office", "wireguard", wgConf)); code != http.StatusOK {
		t.Fatal(code)
	}
	if rev := s.changes.revision(); rev != 0 {
		t.Fatalf("a check announced a change (revision %d)", rev)
	}
	if code := send(h, http.MethodPost, importPath, s.apiToken, importBody("office", "wireguard", wgConf, false)); code != http.StatusCreated {
		t.Fatal(code)
	}
	if rev := s.changes.revision(); rev != 1 {
		t.Fatalf("revision %d after an import", rev)
	}
	recs := sink.all()
	if len(recs) != 2 {
		t.Fatalf("%+v", recs)
	}
	if c := recs[0]; c.Action != "config.profile.import_check" || c.Level != "INFO" || c.Status != http.StatusOK {
		t.Errorf("check: %+v", c)
	}
	if i := recs[1]; i.Action != "config.profile.import" || i.Level != "SECURITY" || i.Status != http.StatusCreated {
		t.Errorf("import: %+v", i)
	}
	for _, r := range recs {
		if strings.Contains(r.Path+r.Name+r.Profile, "PrivateKey") {
			t.Fatalf("body in the audit record: %+v", r)
		}
	}
}
