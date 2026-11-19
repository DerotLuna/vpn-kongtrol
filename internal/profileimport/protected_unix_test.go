//go:build unix

package profileimport

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// A FIFO swapped in after the Lstat check must not hang the root process:
// the open does not block (O_NONBLOCK; a plain open of a FIFO for reading
// waits for a writer) and the fstat refuses it before any read. A final
// symlink is refused by the open itself (O_NOFOLLOW).
func TestOpenRegular_FIFOAndSymlinkDoNotHang(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "swap.conf")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	real := filepath.Join(dir, "real.conf")
	if err := os.WriteFile(real, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.conf")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	type result struct {
		f   *os.File
		err error
	}
	open := func(p string) result {
		done := make(chan result, 1)
		go func() {
			f, _, err := openRegular(p)
			done <- result{f, err}
		}()
		select {
		case r := <-done:
			return r
		case <-time.After(3 * time.Second):
			t.Fatalf("openRegular(%s) blocked", filepath.Base(p))
			return result{}
		}
	}
	for _, p := range []string{fifo, link} {
		if r := open(p); r.err == nil {
			r.f.Close()
			t.Errorf("%s: opened, want refusal", filepath.Base(p))
		}
	}
	r := open(real)
	if r.err != nil {
		t.Fatalf("regular file: %v", r.err)
	}
	r.f.Close()

	if data, err := ReadProtected(fifo); data != nil || key(t, err) != ErrKeyProtectedPath {
		t.Fatalf("FIFO: %q, %v", data, err)
	}
}
