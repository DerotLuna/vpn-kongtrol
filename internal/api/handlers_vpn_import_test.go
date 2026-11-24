package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const wgConf = "[Interface]\nPrivateKey = aaaa\nAddress = 10.0.0.2/32\n\n[Peer]\nPublicKey = bbbb\nAllowedIPs = 10.0.0.0/24\nEndpoint = vpn.example.com:51820\n"

const ovpnConf = "client\ndev tun\nremote vpn.example.com 1194\n<ca>\n-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n</ca>\n"

func importBody(name, typ, content string, overwrite bool) string {
	b, _ := json.Marshal(importReq{Name: name, Type: typ, ContentBase64: base64.StdEncoding.EncodeToString([]byte(content)), Overwrite: overwrite})
	return string(b)
}

func doImport(t *testing.T, s *Server, body string) (int, map[string]any) {
	t.Helper()
	rec := doJSONLang(t, s, http.MethodPost, "/api/v1/vpns/import", "/api/v1/vpns/import", s.handleImportVPNFile, body, "es")
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// Valid files are stored 0600 under the import dir and the path returned;
// no overwrite without the flag.
func TestImportVPNFile_StoresAndNeverOverwritesSilently(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := filepath.Join(t.TempDir(), "configs")
	s.SetImportDir(dir)

	code, out := doImport(t, s, importBody("office", "wireguard", wgConf, false))
	if code != http.StatusCreated || out["path"] != filepath.Join(dir, "office.conf") {
		t.Fatalf("%d %v", code, out)
	}
	fi, err := os.Stat(filepath.Join(dir, "office.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v", fi.Mode().Perm())
	}
	if code, out = doImport(t, s, importBody("office", "wireguard", wgConf, false)); code != http.StatusConflict || out["error_key"] != "api.err.import_exists" {
		t.Fatalf("second import: %d %v", code, out)
	}
	if code, out = doImport(t, s, importBody("office", "wireguard", wgConf+"# v2\n", true)); code != http.StatusCreated || out["overwritten"] != true {
		t.Fatalf("overwrite: %d %v", code, out)
	}
	if code, _ = doImport(t, s, importBody("home", "openvpn", ovpnConf, false)); code != http.StatusCreated {
		t.Fatalf("ovpn: %d", code)
	}
}

// Every unsafe input is refused with a translated error and nothing is
// written: traversal and reserved names, hook scripts, directives that run
// programs or read files (OpenVPN manual: up, down, plugin, …), bad base64.
func TestImportVPNFile_Refusals(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	dir := filepath.Join(t.TempDir(), "configs")
	s.SetImportDir(dir)
	cases := map[string]struct {
		body string
		key  string
	}{
		"traversal":         {importBody("../evil", "wireguard", wgConf, false), "api.err.import_name"},
		"separator":         {importBody(`a\b`, "wireguard", wgConf, false), "api.err.import_name"},
		"hidden":            {importBody(".x", "wireguard", wgConf, false), "api.err.import_name"},
		"reserved":          {importBody("NUL", "wireguard", wgConf, false), "api.err.import_name"},
		"type":              {importBody("x", "ipsec", wgConf, false), "api.err.import_type"},
		"base64":            {`{"name":"x","type":"wireguard","content_base64":"%%%"}`, "api.err.import_content"},
		"wg two interfaces": {importBody("x", "wireguard", wgConf+"[Interface]\n", false), "api.err.import_not_wireguard"},
		"wg hook":           {importBody("x", "wireguard", "[Interface]\nPrivateKey = a\nPostUp = calc.exe\n", false), "api.err.import_hook"},
		"wg unknown key":    {importBody("x", "wireguard", "[Interface]\nPrivateKey = a\nFoo = 1\n", false), "api.err.import_invalid"},
		"ovpn file ref":     {importBody("x", "openvpn", "client\nca /etc/passwd\n", false), "api.err.import_missing_files"},
		"ovpn script-sec 2": {importBody("x", "openvpn", "client\nscript-security 2\n", false), "api.err.import_script_security"},
	}
	for _, d := range []string{"up", "down", "route-up", "ipchange", "tls-verify", "auth-user-pass-verify", "client-connect", "plugin", "learn-address", "log", "config"} {
		cases["ovpn "+d] = struct{ body, key string }{importBody("x", "openvpn", fmt.Sprintf("client\n%s /tmp/x\n", d), false), "api.err.import_directive"}
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			code, out := doImport(t, s, c.body)
			if code != http.StatusBadRequest || out["error_key"] != c.key || out["hint"] == "" {
				t.Fatalf("%d %v", code, out)
			}
		})
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("refused imports wrote files: %v", entries)
	}
}
