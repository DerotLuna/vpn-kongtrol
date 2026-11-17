//go:build windows

package daemonctl

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// executableName is the kongtrol CLI binary's file name on this OS.
const executableName = "kongtrol.exe"

// sysProcAttr detaches the daemon from the calling console: no console
// window at all (DETACHED_PROCESS + CREATE_NO_WINDOW), its own process
// group so a terminal's Ctrl+C never reaches it, and — when breakaway is
// true — out of the caller's job object, so closing the terminal (which
// may kill its whole job) doesn't take the daemon down. The child inherits
// the caller's token: an elevated caller yields an elevated daemon, with
// no UAC prompt here.
func sysProcAttr(breakaway bool) *syscall.SysProcAttr {
	flags := uint32(windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW)
	if breakaway {
		flags |= windows.CREATE_BREAKAWAY_FROM_JOB
	}
	return &syscall.SysProcAttr{CreationFlags: flags, HideWindow: true}
}
