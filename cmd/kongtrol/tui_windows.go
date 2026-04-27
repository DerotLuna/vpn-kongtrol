//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	// Enable ANSI/VT100 escape sequences on Windows 10+.
	// Without this, color codes are printed as literal text in cmd.exe / PowerShell.
	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err == nil {
		_ = windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
