//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// launchBrowser opens url in the OS default browser on Windows.
func launchBrowser(url string) error {
	// The empty argument is required: cmd's "start" treats the first
	// quoted argument as the new window's title, not the target.
	cmd := exec.Command("cmd", "/c", "start", "", url)
	// Put the spawned cmd.exe (and whatever it launches via the shell
	// "start" verb) in its own process group so it stays attached to
	// kongtrol's console but is exempt from Windows' console-control
	// broadcast — otherwise Ctrl+C in the terminal that ran kongtrol can
	// also terminate the browser it opened. Same pattern already used for
	// OpenVPN child processes, see internal/vpn/openvpn/process_windows.go.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}
