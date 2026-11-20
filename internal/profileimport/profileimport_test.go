package profileimport

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

const wgConf = "[Interface]\nPrivateKey = aaaa\nAddress = 10.0.0.2/32\n\n[Peer]\nPublicKey = bbbb\nAllowedIPs = 10.0.0.0/24\nEndpoint = vpn.example.com:51820\n"

const ovpnConf = "client\ndev tun\nremote vpn.example.com 1194\n<ca>\nx\n</ca>\n"

func key(t *testing.T, err error) string {
	t.Helper()
	var ve *config.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want a ValidationError, got %v", err)
	}
	return ve.Key
}

func TestImport_StoresOnceAndOverwritesOnlyWhenAsked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "configs")
	res, err := Import(dir, Request{Name: "office", Type: "WireGuard", Content: []byte(wgConf)})
	if err != nil || res.Path != filepath.Join(dir, "office.conf") || res.Type != "wireguard" || res.Overwritten {
		t.Fatalf("%+v %v", res, err)
	}
	if fi, _ := os.Stat(res.Path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v", fi.Mode().Perm())
	}
	if _, err := Import(dir, Request{Name: "office", Type: "wireguard", Content: []byte(wgConf)}); key(t, err) != "api.err.import_exists" {
		t.Fatal(err)
	}
	res, err = Import(dir, Request{Name: "office", Type: "wireguard", Content: []byte(wgConf + "# v2\n"), Overwrite: true})
	if err != nil || !res.Overwritten {
		t.Fatalf("%+v %v", res, err)
	}
	if b, _ := os.ReadFile(res.Path); !strings.HasSuffix(string(b), "# v2\n") {
		t.Fatalf("not replaced: %q", b)
	}
	if _, err := Import(dir, Request{Name: "home", Type: "openvpn", Content: []byte(ovpnConf)}); err != nil {
		t.Fatal(err)
	}
}

// The reasons come from the real helper validators: a reworded privops
// message fails here instead of silently losing its translation.
func TestImport_RefusalsAreSpecificAndTranslated(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		req  Request
		key  string
		args []any
	}{
		"name":        {Request{Name: "../x", Type: "wireguard", Content: []byte(wgConf)}, "api.err.import_name", nil},
		"reserved":    {Request{Name: "com1", Type: "wireguard", Content: []byte(wgConf)}, "api.err.import_name", nil},
		"type":        {Request{Name: "x", Type: "ipsec", Content: []byte(wgConf)}, "api.err.import_type", nil},
		"empty":       {Request{Name: "x", Type: "wireguard"}, "api.err.import_content", nil},
		"too big":     {Request{Name: "x", Type: "wireguard", Content: make([]byte, MaxSize+1)}, "api.err.import_too_big", nil},
		"wg hook":     {Request{Name: "x", Type: "wireguard", Content: []byte("[Interface]\nPrivateKey = a\npostup = calc.exe\n")}, "api.err.import_hook", []any{"PostUp", 3}},
		"wg shape":    {Request{Name: "x", Type: "wireguard", Content: []byte("[Peer]\nPublicKey = b\n")}, "api.err.import_not_wireguard", nil},
		"wg other":    {Request{Name: "x", Type: "wireguard", Content: []byte("[Interface]\nPrivateKey = a\nFoo = 1\n")}, "api.err.import_invalid", nil},
		"ovpn up":     {Request{Name: "x", Type: "openvpn", Content: []byte("client\nup /tmp/x.sh\n")}, "api.err.import_directive", []any{"up", 2}},
		"ovpn script": {Request{Name: "x", Type: "openvpn", Content: []byte("client\nscript-security 2\n")}, "api.err.import_script_security", []any{2}},
		"ovpn ref":    {Request{Name: "x", Type: "openvpn", Content: []byte("client\nca ca.crt\n")}, "api.err.import_file_ref", []any{"ca.crt"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Import(dir, c.req)
			var ve *config.ValidationError
			if !errors.As(err, &ve) || ve.Key != c.key {
				t.Fatalf("got %v, want %s", err, c.key)
			}
			for i, a := range c.args {
				if ve.Args[i] != a {
					t.Fatalf("args = %v, want %v", ve.Args, c.args)
				}
			}
			for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
				if ve.Message(lang) == "" || ve.Hint(lang) == "" || strings.Contains(ve.Message(lang), "%!") {
					t.Fatalf("%s: %q / %q", lang, ve.Message(lang), ve.Hint(lang))
				}
			}
		})
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("refusals wrote files: %v", entries)
	}
}

func TestTypeForFile(t *testing.T) {
	for in, want := range map[string]string{"a.conf": "wireguard", "B.OVPN": "openvpn", "c.txt": "", "d": ""} {
		if got := TypeForFile(in); got != want {
			t.Fatalf("%s: %q", in, got)
		}
	}
}

// TestTypeForContent: a .conf is WireGuard or OpenVPN by its content
// (OpenVPN GUI on Windows uses .conf, ADR-0080); .ovpn is always OpenVPN.
func TestTypeForContent(t *testing.T) {
	cases := []struct {
		path, content, want string
	}{
		{"wg0.conf", "[Interface]\nPrivateKey = x\n[Peer]\nPublicKey = y\n", "wireguard"},
		{"aws.conf", "client\ndev tun\nproto udp\nremote vpn.example.com 1194\nca ca-aws.crt\n", "openvpn"},
		{"inline.conf", "dev tap\n<ca>\n-----BEGIN CERTIFICATE-----\n</ca>\n", "openvpn"},
		{"CRLF.conf", "client\r\nremote x 1194\r\n", "openvpn"},
		{"mixed.conf", "# remote in a comment\n[Interface]\nremote = nothing\n", "wireguard"},
		{"empty.conf", "", "wireguard"},
		{"x.ovpn", "[Interface]\n", "openvpn"},
		{"x.txt", "client\n", ""},
	}
	for _, c := range cases {
		if got := TypeForContent(c.path, []byte(c.content)); got != c.want {
			t.Errorf("TypeForContent(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
