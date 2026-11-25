package profileimport

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A check runs the whole import and writes nothing (ADR-0118): no folder,
// no file; the same refusals; a taken name is a field, not a refusal.
func TestImport_CheckWritesNothing(t *testing.T) {
	src := t.TempDir()
	write(t, filepath.Join(src, "ca.crt"), caPEM)
	write(t, filepath.Join(src, "keys", "ta.key"), "-----BEGIN OpenVPN Static key V1-----\nab\n-----END OpenVPN Static key V1-----\n")
	dir := filepath.Join(t.TempDir(), "configs")
	content := []byte(ovpnRefs + "auth-user-pass\n")
	res, err := Import(dir, Request{Name: "office", Type: "openvpn", Content: content, Refs: &Refs{Dir: src}, Check: true})
	if err != nil {
		t.Fatal(err)
	}
	want := Result{Type: "openvpn", Included: []string{"ca.crt", "ta.key"}, NeedsCredentials: true, Check: true, File: "office.ovpn"}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("got %+v, want %+v", res, want)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a check created the folder: %v", err)
	}

	// The same refusal as the import, with the same names.
	_, err = Import(dir, Request{Name: "office", Type: "openvpn", Content: []byte(ovpnRefs), Refs: &Refs{Dir: t.TempDir()}, Check: true})
	var miss *MissingFilesError
	if !errors.As(err, &miss) || !reflect.DeepEqual(miss.Names, []string{"ca.crt", "ta.key"}) {
		t.Fatalf("%v", err)
	}
	if _, err := Import(dir, Request{Name: "office", Type: "wireguard", Content: []byte("[Interface]\nPostUp = x\n"), Check: true}); key(t, err) != "api.err.import_hook" {
		t.Fatal(err)
	}

	// A file of that name already stored: reported, not refused, not touched.
	if _, err := Import(dir, Request{Name: "home", Type: "wireguard", Content: []byte(wgConf)}); err != nil {
		t.Fatal(err)
	}
	res, err = Import(dir, Request{Name: "home", Type: "wireguard", Content: []byte(wgConf + "# v2\n"), Check: true})
	if err != nil || !res.Exists || res.Path != "" || res.File != "home.conf" {
		t.Fatalf("%+v %v", res, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "home.conf")); string(b) != wgConf {
		t.Fatalf("a check changed the stored file: %q", b)
	}
	// A folder in the way stays a refusal (the import could never store it).
	if err := os.Mkdir(filepath.Join(dir, "box.conf"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(dir, Request{Name: "box", Type: "wireguard", Content: []byte(wgConf), Check: true}); key(t, err) != "api.err.import_not_file" {
		t.Fatal(err)
	}
}

func TestNeedsCredentials(t *testing.T) {
	for in, want := range map[string]bool{
		"client\nauth-user-pass\n":      true,
		"client\n  auth-user-pass  \n":  true,
		"client\n# auth-user-pass\n":    false,
		"client\nauth-user-pass-verify": false,
		"client\n":                      false,
	} {
		if got := NeedsCredentials("openvpn", []byte(in)); got != want {
			t.Errorf("%q: got %v", in, got)
		}
	}
	if NeedsCredentials("wireguard", []byte("auth-user-pass\n")) {
		t.Error("WireGuard never asks for credentials")
	}
}
