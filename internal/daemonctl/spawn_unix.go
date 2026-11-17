//go:build !windows

package daemonctl

import "syscall"

// executableName is the kongtrol CLI binary's file name on this OS.
const executableName = "kongtrol"

// sysProcAttr starts the daemon in a new session (setsid): no controlling
// terminal, so closing the terminal (SIGHUP) or Ctrl+C in it never reaches
// the daemon. breakaway only matters on Windows.
func sysProcAttr(_ bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
