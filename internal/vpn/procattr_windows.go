//go:build windows

package vpn

import (
	"os/exec"
	"syscall"
)

// HideChildWindow prevents a console window from flashing on screen when
// spawning a console-subsystem helper (VPN client CLIs, powershell, netsh,
// etc.) from a process with no console of its own — notably the system
// tray app, which is built with -H=windowsgui (see Makefile). Without this,
// Windows allocates and briefly shows a new console for every such child,
// which is especially visible for calls made on every monitor.Collector
// tick (e.g. adapters polling interface byte counters via PowerShell).
// No-op on other platforms.
func HideChildWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
