package profileimport

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const caPEM = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"

const ovpnRefs = "client\ndev tun\nremote vpn.example.com 1194\nca ca.crt\ntls-auth keys/ta.key 1\n"

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestImport_ReadsReferencesFromTheConfigFolder(t *testing.T) {
	src := t.TempDir()
	write(t, filepath.Join(src, "ca.crt"), caPEM)
	write(t, filepath.Join(src, "keys", "ta.key"), "-----BEGIN OpenVPN Static key V1-----\nab\n-----END OpenVPN Static key V1-----\n")
	dir := filepath.Join(t.TempDir(), "configs")
	res, err := Import(dir, Request{Name: "office", Type: "openvpn", Content: []byte(ovpnRefs), Refs: &Refs{Dir: src}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Included, []string{"ca.crt", "ta.key"}) {
		t.Fatalf("included = %v", res.Included)
	}
	got, _ := os.ReadFile(res.Path)
	for _, want := range []string{"<ca>\n-----BEGIN CERTIFICATE-----", "</tls-auth>\nkey-direction 1\n"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("stored file lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(string(got), "ca ca.crt") {
		t.Fatalf("reference line kept:\n%s", got)
	}
	// The stored copy is self-contained: the helper's own check passes with no reader.
	if err := Validate("openvpn", got); err != nil {
		t.Fatal(err)
	}
}

func TestImport_MissingReferencesAreAllListed(t *testing.T) {
	dir := t.TempDir()
	_, err := Import(dir, Request{Name: "office", Type: "openvpn", Content: []byte(ovpnRefs + "cert ca.crt\n"), Refs: &Refs{Dir: t.TempDir()}})
	var miss *MissingFilesError
	if !errors.As(err, &miss) || !reflect.DeepEqual(miss.Names, []string{"ca.crt", "ta.key"}) {
		t.Fatalf("%v", err)
	}
	if key(t, err) != ErrKeyMissingFiles {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatal("nothing may be stored while files are missing")
	}
}

func TestImport_AddedFilesMatchByBaseName(t *testing.T) {
	refs, err := NewRefs("", map[string][]byte{
		"CA.CRT":                   []byte(caPEM),
		`C:\Users\ana\Desk\ta.key`: []byte("-----BEGIN OpenVPN Static key V1-----\nab\n-----END OpenVPN Static key V1-----\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(t.TempDir(), Request{Name: "office", Type: "openvpn", Content: []byte(ovpnRefs), Refs: refs})
	if err != nil || len(res.Included) != 2 {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestImport_NeverReadsOutsideTheFolder(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "vpn")
	write(t, filepath.Join(root, "secret.key"), caPEM)
	write(t, filepath.Join(src, "client.ovpn"), "x")
	abs := filepath.Join(root, "secret.key")
	for _, ref := range []string{"../secret.key", `..\secret.key`, abs, "sub/../../secret.key"} {
		conf := "client\nremote a 1\nca \"" + strings.ReplaceAll(ref, `\`, `\\`) + "\"\n"
		_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Dir: src}})
		var miss *MissingFilesError
		if !errors.As(err, &miss) || miss.Names[0] != "secret.key" {
			t.Fatalf("%s: %v", ref, err)
		}
	}
}

func TestImport_LinksFoldersAndBigFilesAreRefused(t *testing.T) {
	src := t.TempDir()
	write(t, filepath.Join(src, "real.crt"), caPEM)
	if err := os.Mkdir(filepath.Join(src, "dir.crt"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(src, "big.crt"), strings.Repeat("a", MaxReferenceSize+1))
	write(t, filepath.Join(src, "bin.crt"), "a\x00b")
	cases := map[string]string{"dir.crt": ErrKeyRefUnreadable, "big.crt": ErrKeyRefTooBig, "bin.crt": ErrKeyRefNotText}
	if os.Symlink(filepath.Join(src, "real.crt"), filepath.Join(src, "link.crt")) == nil {
		cases["link.crt"] = ErrKeyRefUnreadable
	}
	if os.Symlink(src, filepath.Join(src, "linkdir")) == nil {
		cases["linkdir/real.crt"] = ErrKeyRefUnreadable
	}
	for ref, want := range cases {
		conf := "client\nremote a 1\nca " + ref + "\n"
		_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(conf), Refs: &Refs{Dir: src}})
		if key(t, err) != want {
			t.Fatalf("%s: %v, want %s", ref, err, want)
		}
	}
}

func TestNewRefs_Limits(t *testing.T) {
	many := map[string][]byte{}
	for i := range MaxExtraFiles + 1 {
		many[string(rune('a'+i))+".crt"] = []byte("x")
	}
	if _, err := NewRefs("", many); key(t, err) != ErrKeyTooManyFiles {
		t.Fatal(err)
	}
	if _, err := NewRefs("", map[string][]byte{"a.crt": make([]byte, MaxReferenceSize+1)}); key(t, err) != ErrKeyRefTooBig {
		t.Fatal(err)
	}
	if _, err := NewRefs("", map[string][]byte{"..": []byte("x")}); key(t, err) != "api.err.import_content" {
		t.Fatal(err)
	}
}

func TestImport_WithoutRefsStaysSelfContainedOnly(t *testing.T) {
	_, err := Import(t.TempDir(), Request{Name: "o", Type: "openvpn", Content: []byte(ovpnRefs)})
	if key(t, err) != "api.err.import_file_ref" {
		t.Fatal(err)
	}
}
