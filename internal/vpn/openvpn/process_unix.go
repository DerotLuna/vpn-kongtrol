//go:build linux || darwin

package openvpn

import (
	"os/exec"
	"syscall"
)

// setProcAttr configures the command so the child process belongs to its own
// process group. This ensures SIGTERM reaches the entire subprocess tree.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
