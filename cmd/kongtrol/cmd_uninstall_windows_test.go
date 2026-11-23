//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// Regression: programs resolved by name searched the current directory
// first (a planted ping.exe ran). PowerShell is taken by full path, starts
// in System32 and the path is not part of its command line (ADR-0100).
func TestDeleteAfterExitCmd_FullPathsAndNoPathInCommand(t *testing.T) {
	sys := systemDirectory()
	target := `C:\Users\x\AppData\Local\kongtrol\bin\kongtrol.exe`
	cmd := deleteAfterExitCmd(target, 42)
	if want := filepath.Join(sys, "WindowsPowerShell", "v1.0", "powershell.exe"); !strings.EqualFold(cmd.Path, want) {
		t.Errorf("program = %q, want %q", cmd.Path, want)
	}
	if !strings.EqualFold(cmd.Dir, sys) {
		t.Errorf("dir = %q, want System32", cmd.Dir)
	}
	for _, a := range cmd.Args {
		if strings.Contains(a, target) {
			t.Errorf("path leaked into the command line: %q", a)
		}
	}
	if strings.Contains(deleteAfterExitScript, `"`) {
		t.Error("script must not contain double quotes")
	}
	env := strings.Join(cmd.Env, "\n")
	if !strings.Contains(env, deletePathEnv+"="+target) || !strings.Contains(env, deleteWaitEnv+"=42") {
		t.Errorf("inputs missing from the environment")
	}
}

// The delayed delete removes exactly the named file, whatever its path
// holds: spaces, %VAR% (cmd.exe expanded it), &, ^, ! and wildcard
// brackets. A file at the path %USERNAME% would expand to stays.
func TestDeleteAfterExitCmd_DeletesLiteralPath(t *testing.T) {
	user := os.Getenv("USERNAME")
	if user == "" {
		t.Skip("USERNAME not set")
	}
	base := t.TempDir()
	literal := filepath.Join(base, "a %USERNAME% & ^b! [1]")
	expanded := filepath.Join(base, "a "+user+" & ^b! [1]")
	for _, d := range []string{literal, expanded} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "kongtrol.exe"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := deleteAfterExitCmd(filepath.Join(literal, "kongtrol.exe"), 0)
	cmd.SysProcAttr.CreationFlags = 0
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(literal, "kongtrol.exe")); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(expanded, "kongtrol.exe")); err != nil {
		t.Fatalf("deleted the expanded path instead: %v", err)
	}
}

// It waits for the given process before deleting.
func TestDeleteAfterExitCmd_WaitsForProcess(t *testing.T) {
	f := filepath.Join(t.TempDir(), "kongtrol.exe")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// A child that lives ~2 s (ping waits 1 s between echoes).
	hold := exec.Command(filepath.Join(systemDirectory(), "PING.EXE"), "-n", "3", "127.0.0.1")
	if err := hold.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = hold.Wait() }()
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(hold.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	start := time.Now()
	cmd := deleteAfterExitCmd(f, hold.Process.Pid)
	cmd.SysProcAttr.CreationFlags = 0
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if ev, _ := windows.WaitForSingleObject(h, 0); ev != windows.WAIT_OBJECT_0 {
		t.Fatalf("delete finished while the process was still running (after %v)", time.Since(start))
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
}
