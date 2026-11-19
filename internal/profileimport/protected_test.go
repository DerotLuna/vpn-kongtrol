package profileimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
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
	cases := map[string]struct {
		path string
		key  string
	}{
		"relative":      {"wg0.conf", ErrKeyProtectedPath},
		"other ext":     {write("shadow", secret), ErrKeyProtectedPath},
		"folder":        {mkdir(t, filepath.Join(dir, "d.conf")), ErrKeyProtectedPath},
		"missing":       {filepath.Join(dir, "nope.conf"), ErrKeyProtectedPath},
		"not wireguard": {write("shadow.conf", secret), ""},
		"wg with hook":  {write("hook.conf", "[Interface]\nPrivateKey = a\nPostUp = id\n"), "api.err.import_hook"},
		"too big":       {write("big.conf", strings.Repeat("#", int(MaxSize)+1)), "api.err.import_too_big"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := ReadProtected(c.path)
			if err == nil || data != nil {
				t.Fatalf("want refusal and no content, got %q, %v", data, err)
			}
			if c.key != "" && key(t, err) != c.key {
				t.Fatalf("key = %q, want %q", key(t, err), c.key)
			}
			if strings.Contains(err.Error(), "$6$hash") {
				t.Fatalf("refusal echoes the file content: %v", err)
			}
		})
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
