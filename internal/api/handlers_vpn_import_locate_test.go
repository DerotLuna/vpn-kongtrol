package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/filepicker"
)

// Tests never walk the real home folder (ADR-0112).
func init() { importSearchRoots = func() []string { return []string{} } }

func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

const ovpnTwoRefs = "client\ndev tun\nremote vpn.example.com 1194\nca ca.crt\ncert rluna.crt\n"

func writeAll(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// The owner's case: rluna.conf dropped in the browser (no path) while the
// original sits next to its certificates in OpenVPN\config. The daemon
// finds the identical original and answers where (found_in).
func TestImportVPNFile_DropFindsTheOriginal(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	root := realDir(t)
	src := filepath.Join(root, "OpenVPN", "config")
	writeAll(t, src, map[string]string{"rluna.conf": ovpnTwoRefs, "ca.crt": caPEM, "rluna.crt": caPEM})
	old := importSearchRoots
	importSearchRoots = func() []string { return []string{root} }
	t.Cleanup(func() { importSearchRoots = old })

	body, _ := json.Marshal(importReq{Name: "rluna", Type: "openvpn", FileName: "rluna.conf", ContentBase64: b64(ovpnTwoRefs)})
	code, out := doImport(t, s, string(body))
	if code != http.StatusCreated || out["found_in"] != src {
		t.Fatalf("%d %v", code, out)
	}
	if inc, _ := out["included"].([]any); len(inc) != 2 {
		t.Fatalf("included = %v", out["included"])
	}
}

// "Elegir la carpeta donde están": the folder picked is where the files are read.
func TestImportVPNFile_RefDir(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	certs := realDir(t)
	writeAll(t, certs, map[string]string{"ca.crt": caPEM, "rluna.crt": caPEM})
	body, _ := json.Marshal(importReq{Name: "rluna", Type: "openvpn", ContentBase64: b64(ovpnTwoRefs), RefDir: certs})
	if code, out := doImport(t, s, string(body)); code != http.StatusCreated || out["found_in"] != nil {
		t.Fatalf("%d %v", code, out)
	}
	for _, bad := range []string{"relative", `\\host\share`, filepath.Join(certs, "ca.crt")} {
		body, _ := json.Marshal(importReq{Name: "rluna2", Type: "openvpn", ContentBase64: b64(ovpnTwoRefs), RefDir: bad})
		if code, out := doImport(t, s, string(body)); code != http.StatusBadRequest || out["error_key"] != "api.err.import_protected_path" {
			t.Fatalf("%s: %d %v", bad, code, out)
		}
	}
}

// An absolute path in a dropped .ovpn is read (the daemon runs as the user).
func TestImportVPNFile_AbsoluteReference(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	keys := realDir(t)
	writeAll(t, keys, map[string]string{"ca.crt": caPEM})
	conf := "client\nremote a 1\nca \"" + strings.ReplaceAll(filepath.Join(keys, "ca.crt"), `\`, `\\`) + "\"\n"
	body, _ := json.Marshal(importReq{Name: "abs", Type: "openvpn", ContentBase64: b64(conf)})
	if code, out := doImport(t, s, string(body)); code != http.StatusCreated {
		t.Fatalf("%d %v", code, out)
	}
}

func doPickFolder(t *testing.T, s *Server) (int, map[string]any) {
	t.Helper()
	rec := doJSONLang(t, s, http.MethodPost, importPickFolderPath, importPickFolderPath, s.handleImportPickFolder, "", "en")
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestImportPickFolder(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := realDir(t)
	var gotTitle string
	folder := func(path string, err error) Picker {
		return Picker{Available: func() bool { return true }, PickFolder: func(_ context.Context, o filepicker.Options) (string, error) {
			gotTitle = o.Title
			return path, err
		}}
	}
	s.SetPicker(folder(dir, nil))
	if code, out := doPickFolder(t, s); code != http.StatusOK || out["path"] != dir || out["name"] != filepath.Base(dir) {
		t.Fatalf("%d %v", code, out)
	}
	if gotTitle != "Kongtrol · Choose the folder with the .ovpn's certificates" {
		t.Fatalf("title = %q", gotTitle)
	}
	s.SetPicker(folder("", filepicker.ErrCanceled))
	if code, out := doPickFolder(t, s); code != http.StatusOK || out["cancelled"] != true {
		t.Fatalf("%d %v", code, out)
	}
	s.SetPicker(folder("", filepicker.ErrBusy))
	if code, out := doPickFolder(t, s); code != http.StatusConflict || out["error_key"] != "api.err.picker_busy" {
		t.Fatalf("%d %v", code, out)
	}
	s.SetPicker(folder("", filepicker.ErrUnavailable))
	if code, out := doPickFolder(t, s); code != http.StatusServiceUnavailable || out["error_key"] != "api.err.picker_unavailable" {
		t.Fatalf("%d %v", code, out)
	}
	file := filepath.Join(dir, "x.txt")
	_ = os.WriteFile(file, []byte("x"), 0o600)
	s.SetPicker(folder(file, nil))
	if code, out := doPickFolder(t, s); code != http.StatusBadRequest || out["error_key"] != "api.err.import_protected_path" {
		t.Fatalf("%d %v", code, out)
	}
}
