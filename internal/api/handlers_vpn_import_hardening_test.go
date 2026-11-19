package api

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The "path" import takes a file the signed-in user picked; only a clean
// absolute path to a regular file is read (CodeQL go/path-injection review).
func TestImportVPNFile_PathRefusals(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	src := t.TempDir()
	good := filepath.Join(src, "wg0.conf")
	if err := os.WriteFile(good, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	dirConf := filepath.Join(src, "dir.conf")
	if err := os.Mkdir(dirConf, 0o700); err != nil {
		t.Fatal(err)
	}
	refused := map[string]string{
		"relative":      "wg0.conf",
		"dot-dot":       filepath.Join(src, "sub") + string(filepath.Separator) + ".." + string(filepath.Separator) + "wg0.conf",
		"directory":     dirConf,
		"wrong type":    filepath.Join(src, "wg0.txt"),
		"nul":           good + "\x00.conf",
		"unix device":   "/dev/null.conf",
		"proc":          "/proc/self/environ.conf",
		"win device":    `\\.\COM1.conf`,
		"win long path": `\\?\C:\x\wg0.conf`,
		"unc share":     `\\server\share\wg0.conf`,
	}
	link := filepath.Join(src, "link.conf")
	if err := os.Symlink(good, link); err == nil {
		refused["symlink"] = link
	} else if runtime.GOOS != "windows" {
		t.Fatalf("symlink: %v", err)
	} // Windows without the symlink privilege: covered by the Lstat check.
	for name, p := range refused {
		code, out := doImport(t, s, pathBody(p, false))
		if code != http.StatusBadRequest || out["error_key"] != "api.err.import_protected_path" {
			t.Errorf("%s (%q): %d %v", name, p, code, out)
		}
	}
	if code, out := doImport(t, s, pathBody(good, false)); code != http.StatusCreated {
		t.Fatalf("regular file: %d %v", code, out)
	}
}

func TestCleanImportPath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a.conf")
	if got, ok := cleanImportPath("  " + abs + " "); !ok || got != abs {
		t.Fatalf("clean absolute: %q %v", got, ok)
	}
	for _, p := range []string{"", "a.conf", "./a.conf", `..\a.conf`, "/dev", "/sys/x.conf",
		// NT object paths: absolute for Windows, and they reach SMB or devices.
		`\??\UNC\srv\sh\a.ovpn`, `\??\C:\x\a.conf`, `\??\GLOBALROOT\Device\Mup\srv\sh\a.conf`,
		`/??/UNC/srv/sh/a.conf`, `\\?\C:\x\a.conf`, `\\.\C:\x\a.conf`, `\\srv\sh\a.conf`,
		`//srv/sh/a.conf`, `\\?\Volume{0b1c2d3e-0000-0000-0000-000000000000}\a.conf`,
	} {
		if _, ok := cleanImportPath(p); ok {
			t.Errorf("%q accepted", p)
		}
	}
	if runtime.GOOS == "windows" {
		if _, ok := cleanImportPath(`C:\Users\x\wg0.conf`); !ok {
			t.Error("a drive-letter path must be accepted")
		}
		if _, ok := cleanImportPath(`\Users\x\wg0.conf`); ok {
			t.Error("a path without a drive letter must be refused")
		}
	}
}

// A swap between the check and the read (file replaced by another one) is
// refused: the descriptor read must be the file checked.
func TestReadRegularFileRefusesSwap(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.conf")
	if err := os.WriteFile(p, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := readRegularFile(p); err != nil || string(b) != wgConf {
		t.Fatalf("read: %q %v", b, err)
	}
	if _, err := readRegularFile(dir); err == nil {
		t.Fatal("directory read")
	}
	big := filepath.Join(dir, "big.conf")
	if err := os.WriteFile(big, []byte(strings.Repeat("x", 2<<20)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularFile(big); err == nil {
		t.Fatal("over the size cap")
	}
}

// The elevated read waits for an OS prompt without the config lock, so
// other changes go on; the lock is taken only to store the file.
func TestImportElevatedReadDoesNotHoldConfigLock(t *testing.T) {
	if isConfigMutation(importPath) {
		t.Fatal("secureHandler must not lock the config for the import route")
	}
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))
	src := filepath.Join(t.TempDir(), "wg0.conf")
	if err := os.WriteFile(src, []byte(wgConf), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := importReadFile
	t.Cleanup(func() { importReadFile = orig })
	importReadFile = func(string) ([]byte, error) { return nil, &fs.PathError{Op: "open", Err: fs.ErrPermission} }
	waiting, release := make(chan struct{}), make(chan struct{})
	s.SetProtectedReader(func(context.Context, string) ([]byte, error) {
		close(waiting)
		<-release
		return []byte(wgConf), nil
	})
	done := make(chan int)
	go func() {
		code, _ := doImport(t, s, pathBody(src, true))
		done <- code
	}()
	<-waiting
	if !s.configMu.TryLock() {
		t.Fatal("config lock held while the permission prompt is open")
	}
	s.configMu.Unlock()
	close(release)
	select {
	case code := <-done:
		if code != http.StatusCreated {
			t.Fatalf("elevated import: %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("import did not finish")
	}
}
