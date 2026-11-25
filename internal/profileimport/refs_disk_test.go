package profileimport

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func quoteRef(p string) string { return "\"" + strings.ReplaceAll(p, `\`, `\\`) + "\"" }

// With Disk (the daemon, the CLI) an absolute path and a "../" leaving the
// folder are read like OpenVPN GUI would, also for a dropped file with no
// folder (ADR-0112).
func TestImport_ReadsAbsoluteAndParentReferences(t *testing.T) {
	root := realTemp(t)
	src := filepath.Join(root, "vpn")
	write(t, filepath.Join(root, "keys", "ca.crt"), caPEM)
	write(t, filepath.Join(src, "client.ovpn"), "x")
	abs := filepath.Join(root, "keys", "ca.crt")
	for _, c := range []struct{ dir, ref string }{{src, abs}, {"", abs}, {src, "../keys/ca.crt"}, {src, "sub/../../keys/ca.crt"}} {
		conf := "client\nremote a 1\nca " + quoteRef(c.ref) + "\n"
		res, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Dir: c.dir, Disk: true}})
		if err != nil || len(res.Included) != 1 {
			t.Fatalf("%+v: %+v %v", c, res, err)
		}
		got, _ := os.ReadFile(res.Path)
		if !strings.Contains(string(got), "<ca>\n-----BEGIN CERTIFICATE-----") {
			t.Fatalf("%+v: stored\n%s", c, got)
		}
	}
}

func TestImport_OutsideReferencesMustBeKeys(t *testing.T) {
	root := realTemp(t)
	sshKey := filepath.Join(root, "id_ed25519")
	write(t, sshKey, "-----BEGIN OPENSSH PRIVATE KEY-----\nb3Bl\n-----END OPENSSH PRIVATE KEY-----\n")
	notes := filepath.Join(root, "notes.txt")
	write(t, notes, "aws_secret_access_key = x\n")
	mixed := filepath.Join(root, "mixed.pem")
	write(t, mixed, caPEM+"-----BEGIN OPENSSH PRIVATE KEY-----\nb3Bl\n-----END OPENSSH PRIVATE KEY-----\n")
	for _, p := range []string{sshKey, notes, mixed} {
		conf := "client\nremote a 1\ncert " + quoteRef(p) + "\n"
		_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Disk: true}})
		if key(t, err) != ErrKeyRefNotKey {
			t.Fatalf("%s: %v", p, err)
		}
	}
	// Inside the .ovpn's own folder the content rule stays the helper's.
	write(t, filepath.Join(root, "inside.crt"), "plain text cert\n")
	if _, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte("client\nremote a 1\nca inside.crt\n"), Refs: &Refs{Dir: root, Disk: true}}); err != nil {
		t.Fatal(err)
	}
}

func TestImport_OutsideReferencesNeverReachTheNetworkOrLinks(t *testing.T) {
	root := realTemp(t)
	write(t, filepath.Join(root, "real", "ca.crt"), caPEM)
	refs := []string{`\\attacker.example\share\ca.crt`, `//attacker.example/share/ca.crt`, "/proc/self/environ"}
	if runtime.GOOS == "windows" {
		refs = append(refs, `\\?\C:\x\ca.crt`, `\??\UNC\host\s\ca.crt`)
	}
	for _, ref := range refs {
		conf := "client\nremote a 1\nca " + quoteRef(ref) + "\n"
		_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Disk: true}})
		var miss *MissingFilesError
		if !errors.As(err, &miss) {
			t.Fatalf("%s: %v", ref, err)
		}
	}
	if os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")) == nil {
		conf := "client\nremote a 1\nca " + quoteRef(filepath.Join(root, "link", "ca.crt")) + "\n"
		_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Disk: true}})
		if key(t, err) != ErrKeyRefUnreadable {
			t.Fatalf("through a link: %v", err)
		}
	}
}

func TestLooksLikeKeyFile(t *testing.T) {
	ok := []string{caPEM, tlsKey, "Bag Attributes\n" + caPEM, "-----BEGIN ENCRYPTED PRIVATE KEY-----\r\nx\r\n-----END ENCRYPTED PRIVATE KEY-----\r\n"}
	bad := []string{"", "hello", "-----BEGIN OPENSSH PRIVATE KEY-----\nx\n", "-----BEGIN PGP PRIVATE KEY BLOCK-----\nx\n"}
	for _, s := range ok {
		if !LooksLikeKeyFile([]byte(s)) {
			t.Errorf("refused %q", s)
		}
	}
	for _, s := range bad {
		if LooksLikeKeyFile([]byte(s)) {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestReferencedPaths(t *testing.T) {
	got := ReferencedPaths([]byte(ovpnOwner + "<key>\nx\n</key>\n"))
	if strings.Join(got, ",") != "ca.crt,ta.key" {
		t.Fatalf("%v", got)
	}
	if ReferencedPaths([]byte("up /bin/sh\n")) != nil {
		t.Fatal("refused config listed references")
	}
}
