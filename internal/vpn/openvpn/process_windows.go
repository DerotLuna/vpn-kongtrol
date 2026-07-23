//go:build windows

package openvpn

import (
	"os/exec"
	"syscall"
)

// setProcAttr configures the command so the child process is in its own
// process group (CREATE_NEW_PROCESS_GROUP), so Ctrl+C events don't propagate
// to the child unintentionally, and with no console window of its own
// (HideWindow) — openvpn.exe is a console-subsystem binary, and a parent
// with no console (the tray, built with -H=windowsgui) would otherwise get
// a visible console window for the tunnel's entire lifetime.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}
