//go:build windows

package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestCurrentDNSWindows_TimesOutOnHungNetsh proves the netsh call is
// actually bounded by netshQueryTimeout instead of blocking forever when
// the subprocess never exits on its own (e.g. netsh wedged on a busy or
// unusual system).
//
// The fake "netsh" must be a real single native process (not a .bat/.cmd
// script) — Go's os/exec on Windows runs .bat/.cmd through a wrapping
// cmd.exe, and killing that wrapper on timeout does not kill any further
// child process it spawned (no Job Object), which would make this test
// exercise an unrelated cmd.exe/child-process quirk instead of the
// CommandContext timeout behavior this is meant to verify.
func TestCurrentDNSWindows_TimesOutOnHungNetsh(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "hang.go")
	if err := os.WriteFile(src, []byte("package main\nimport \"time\"\nfunc main() { time.Sleep(time.Hour) }\n"), 0o644); err != nil {
		t.Fatalf("write helper source: %v", err)
	}
	fakeNetsh := filepath.Join(tmpDir, "netsh.exe")
	build := exec.Command("go", "build", "-o", fakeNetsh, src)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake netsh helper: %v\n%s", err, out)
	}

	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
	if err := os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	start := time.Now()
	_, err := currentDNSWindows("eth0")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a timed-out netsh call, got nil")
	}
	if elapsed >= netshQueryTimeout+3*time.Second {
		t.Fatalf("currentDNSWindows took %v — expected it to be cut off around the %v timeout, not hang", elapsed, netshQueryTimeout)
	}
	t.Logf("returned after %v (timeout=%v): %v", elapsed, netshQueryTimeout, err)
}
