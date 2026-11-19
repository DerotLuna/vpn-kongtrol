package profileimport

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestReadProtected_ReleasesOnlyValidProfiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := write("wg0.conf", wgConf)
	got, err := ReadProtected(good)
	if err != nil || string(got) != wgConf {
		t.Fatalf("valid WireGuard config: got %q, %v", got, err)
	}
	if _, err := ReadProtected(write("vpn.ovpn", ovpnConf)); err != nil {
		t.Fatalf("valid .ovpn: %v", err)
	}

	secret := "root:$6$hash:19000:0:99999:7:::\n"
	cases := map[string]string{
		"relative":       "wg0.conf",
		"other ext":      write("shadow", secret),
		"folder":         mkdir(t, filepath.Join(dir, "d.conf")),
		"missing":        filepath.Join(dir, "nope.conf"),
		"not wireguard":  write("shadow.conf", secret),
		"not openvpn":    write("shadow.ovpn", secret),
		"wg with hook":   write("hook.conf", "[Interface]\nPrivateKey = a\nPostUp = id\n"),
		"wg odd key":     write("odd.conf", "[Interface]\nPrivateKey = a\nRootSecret = s3cr3t\n"),
		"wg odd section": write("sect.conf", "[Interface]\nPrivateKey = a\n[s3cr3t]\n"),
		"ovpn bad line":  write("bad.ovpn", "client\nremote vpn.example.com\n</s3cr3t>\n"),
		"ovpn no remote": write("noremote.ovpn", "client\ndev tun\n"),
		"ovpn no client": write("noclient.ovpn", "dev tun\nremote vpn.example.com 1194\n"),
		"ovpn remote only in a block": write("inblock.ovpn",
			"client\n<ca>\nremote vpn.example.com\n</ca>\n"),
		"too big": write("big.conf", strings.Repeat("#", int(MaxSize)+1)),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := ReadProtected(path)
			if err == nil || data != nil {
				t.Fatalf("want refusal and no content, got %q, %v", data, err)
			}
			// One generic refusal: it never says which check failed nor
			// echoes a section, key or line of the file (ADR-0063).
			if key(t, err) != ErrKeyProtectedPath {
				t.Fatalf("key = %q, want %q", key(t, err), ErrKeyProtectedPath)
			}
			for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
				msg, hint, _ := config.LocalizeError(err, lang)
				for _, leak := range []string{"$6$hash", "s3cr3t", "RootSecret", "PostUp"} {
					if strings.Contains(err.Error()+msg+hint, leak) {
						t.Fatalf("refusal echoes the file: %q / %q / %v", msg, hint, err)
					}
				}
			}
		})
	}
}

// A client profile in either shape (top-level remote, or remote inside
// <connection>; client or tls-client) is released.
func TestReadProtected_OpenVPNShapes(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"client.ovpn":     ovpnConf,
		"tlsclient.ovpn":  "tls-client\npull\ndev tun\nremote 198.51.100.7 1194 udp\n",
		"connection.ovpn": "--client\ndev tun\n<connection>\nremote vpn.example.com 443 tcp\n</connection>\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, err := ReadProtected(p); err != nil || string(got) != body {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
}

// A hard link gives a root-only file a second name (e.g. a user-made
// link.ovpn in a folder with fs.protected_hardlinks off): refused.
func TestReadProtected_RefusesHardLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("link count is checked on Unix only (no root-only profile files on Windows)")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real.conf")
	if err := os.WriteFile(target, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "second.conf")
	if err := os.Link(target, link); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	for _, p := range []string{target, link} {
		if data, err := ReadProtected(p); data != nil || key(t, err) != ErrKeyProtectedPath {
			t.Fatalf("%s with 2 links: %q, %v", p, data, err)
		}
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProtected(target); err != nil {
		t.Fatalf("single link again: %v", err)
	}
}

func TestReadProtected_RefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.conf")
	if err := os.WriteFile(target, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadProtected(link); key(t, err) != ErrKeyProtectedPath {
		t.Fatalf("symlink: %v", err)
	}
}

func TestCheckFileReadable_MissingIsNotRootOnly(t *testing.T) {
	if err := config.CheckFileReadable(filepath.Join(t.TempDir(), "x.conf")); err != nil {
		t.Fatalf("missing file: %v", err)
	}
	if err := config.CheckFileReadable(""); err != nil {
		t.Fatalf("empty: %v", err)
	}
}

func mkdir(t *testing.T, p string) string {
	t.Helper()
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}
