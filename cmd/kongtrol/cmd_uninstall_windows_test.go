//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression: "ping" was resolved by cmd.exe, which searches the current
// directory first (a planted ping.exe next to the user's shell ran).
func TestDeleteAfterExitCmd_FullPaths(t *testing.T) {
	sys := systemDirectory()
	cmd := deleteAfterExitCmd(`C:\Users\x\AppData\Local\kongtrol\bin\kongtrol.exe`)
	if !strings.EqualFold(cmd.Path, filepath.Join(sys, "cmd.exe")) {
		t.Errorf("cmd path = %q", cmd.Path)
	}
	if !strings.EqualFold(cmd.Dir, sys) {
		t.Errorf("dir = %q, want System32", cmd.Dir)
	}
	line := cmd.SysProcAttr.CmdLine
	if !strings.Contains(line, `/d /c ""`+filepath.Join(sys, "PING.EXE")+`" -n 4`) {
		t.Errorf("ping not by full path: %q", line)
	}
}

// The delayed delete really removes a file whose path has spaces.
func TestDeleteAfterExitCmd_DeletesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "kongtrol.exe")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := deleteAfterExitCmd(f)
	cmd.SysProcAttr.CreationFlags = 0
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
}
