//go:build windows

package openvpn

import (
	"os/exec"
	"syscall"
)

// setProcAttr configures the command so the child process is in its own
// process group (CREATE_NEW_PROCESS_GROUP). On Windows this is needed so
// Ctrl+C events don't propagate to the child unintentionally.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
