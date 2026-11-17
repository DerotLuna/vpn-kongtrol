//go:build linux || darwin

package privops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTrustedFileRejectsWritableByOthers(t *testing.T) {
	p := filepath.Join(t.TempDir(), "shared.ovpn")
	if err := os.WriteFile(p, []byte("client\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readTrustedFile(selfCaller(), p, 100, nil); !errors.Is(err, ErrDenied) {
		t.Errorf("world-writable file: %v", err)
	}
	// A different (non-root) caller does not own the file.
	_ = os.Chmod(p, 0o644)
	other := Caller{ID: "4242424", UID: 4242424}
	if os.Geteuid() != 0 {
		if _, _, err := readTrustedFile(other, p, 100, nil); !errors.Is(err, ErrDenied) {
			t.Errorf("foreign owner accepted: %v", err)
		}
	}
	// Symlinks are resolved; the target is what gets checked.
	link := filepath.Join(t.TempDir(), "link.ovpn")
	if err := os.Symlink(p, link); err != nil {
		t.Skip(err)
	}
	if _, resolved, err := readTrustedFile(selfCaller(), link, 100, nil); err != nil || resolved == link {
		t.Errorf("symlink: %v %s", err, resolved)
	}
}
