package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/filepicker"
)

const caPEM = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"

const ovpnWithRefs = "client\ndev tun\nremote vpn.example.com 1194\nca ca.crt\n"

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// An .ovpn picked with a real path reads "ca ca.crt" from its own folder and
// stores a self-contained copy (ADR-0105).
func TestImportVPNFile_PathReadsReferencesFromItsFolder(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := filepath.Join(t.TempDir(), "configs")
	s.SetImportDir(dir)
	src := t.TempDir()
	conf := filepath.Join(src, "office.ovpn")
	if err := os.WriteFile(conf, []byte(ovpnWithRefs), 0o600); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(importReq{Path: conf})

	// Not there yet: the refusal names the file, so the panel offers "Agregar ca.crt".
	code, out := doImport(t, s, string(body))
	if code != http.StatusBadRequest || out["error_key"] != "api.err.import_missing_files" {
		t.Fatalf("%d %v", code, out)
	}
	if m, _ := out["missing"].([]any); len(m) != 1 || m[0] != "ca.crt" {
		t.Fatalf("missing = %v", out["missing"])
	}

	if err := os.WriteFile(filepath.Join(src, "ca.crt"), []byte(caPEM), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out = doImport(t, s, string(body))
	if code != http.StatusCreated {
		t.Fatalf("%d %v", code, out)
	}
	if inc, _ := out["included"].([]any); len(inc) != 1 || inc[0] != "ca.crt" {
		t.Fatalf("included = %v", out["included"])
	}
	stored, _ := os.ReadFile(filepath.Join(dir, "office.ovpn"))
	if !strings.Contains(string(stored), "<ca>\n-----BEGIN CERTIFICATE-----") || strings.Contains(string(stored), "ca ca.crt") {
		t.Fatalf("stored:\n%s", stored)
	}
}

// The browser cannot give a path: the files dropped with the .ovpn (or
// added with "Agregar") are matched by base name.
func TestImportVPNFile_AddedFiles(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	body, _ := json.Marshal(importReq{
		Name: "office", Type: "openvpn", ContentBase64: b64(ovpnWithRefs),
		Files: []importFile{{Name: "ca.crt", ContentBase64: b64(caPEM)}},
	})
	if code, out := doImport(t, s, string(body)); code != http.StatusCreated {
		t.Fatalf("%d %v", code, out)
	}
	many := make([]importFile, 9)
	for i := range many {
		many[i] = importFile{Name: string(rune('a'+i)) + ".crt", ContentBase64: b64(caPEM)}
	}
	body, _ = json.Marshal(importReq{Name: "x", Type: "openvpn", ContentBase64: b64(ovpnWithRefs), Files: many})
	if code, out := doImport(t, s, string(body)); code != http.StatusRequestEntityTooLarge || out["error_key"] != "api.err.import_too_many_files" {
		t.Fatalf("%d %v", code, out)
	}
	body, _ = json.Marshal(importReq{Name: "x", Type: "openvpn", ContentBase64: b64(ovpnWithRefs), Files: []importFile{{Name: "ca.crt", ContentBase64: "%%"}}})
	if code, out := doImport(t, s, string(body)); code != http.StatusBadRequest || out["error_key"] != "api.err.import_content" {
		t.Fatalf("%d %v", code, out)
	}
}

func doPick(t *testing.T, s *Server, method string) (int, map[string]any) {
	t.Helper()
	h := s.handleImportPick
	if method == http.MethodGet {
		h = s.handleImportPickAvailable
	}
	rec := doJSONLang(t, s, method, importPickPath, importPickPath, h, "", "en")
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestImportPick(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	src := t.TempDir()
	conf := filepath.Join(src, "office.ovpn")
	if err := os.WriteFile(conf, []byte(ovpnWithRefs), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotTitle string
	pick := func(path string, err error) func(context.Context, filepicker.Options) (string, error) {
		return func(_ context.Context, o filepicker.Options) (string, error) {
			gotTitle = o.Title
			return path, err
		}
	}

	s.SetPicker(Picker{Available: func() bool { return false }, Pick: pick("", filepicker.ErrUnavailable)})
	if code, out := doPick(t, s, http.MethodGet); code != http.StatusOK || out["available"] != false {
		t.Fatalf("%d %v", code, out)
	}
	if code, out := doPick(t, s, http.MethodPost); code != http.StatusServiceUnavailable || out["error_key"] != "api.err.picker_unavailable" {
		t.Fatalf("%d %v", code, out)
	}

	s.SetPicker(Picker{Available: func() bool { return true }, Pick: pick(conf, nil)})
	code, out := doPick(t, s, http.MethodPost)
	if code != http.StatusOK || out["path"] != conf || out["name"] != "office.ovpn" || out["type"] != "openvpn" || out["content_base64"] != b64(ovpnWithRefs) {
		t.Fatalf("%d %v", code, out)
	}
	if gotTitle != "Kongtrol · Choose your VPN file" {
		t.Fatalf("title = %q", gotTitle)
	}

	s.SetPicker(Picker{Available: func() bool { return true }, Pick: pick("", filepicker.ErrCanceled)})
	if code, out := doPick(t, s, http.MethodPost); code != http.StatusOK || out["cancelled"] != true {
		t.Fatalf("%d %v", code, out)
	}
	s.SetPicker(Picker{Available: func() bool { return true }, Pick: pick("", filepicker.ErrBusy)})
	if code, out := doPick(t, s, http.MethodPost); code != http.StatusConflict || out["error_key"] != "api.err.picker_busy" {
		t.Fatalf("%d %v", code, out)
	}
	// A picked file that is not a VPN file name is refused like a "path" import.
	other := filepath.Join(src, "notes.txt")
	_ = os.WriteFile(other, []byte("x"), 0o600)
	s.SetPicker(Picker{Available: func() bool { return true }, Pick: pick(other, nil)})
	if code, out := doPick(t, s, http.MethodPost); code != http.StatusBadRequest || out["error_key"] != "api.err.import_protected_path" {
		t.Fatalf("%d %v", code, out)
	}
}
