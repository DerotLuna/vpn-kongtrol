//go:build darwin

package openvpn

import (
	"os/exec"
	"syscall"
)

// lifetimeHandle is unused on macOS.
type lifetimeHandle = struct{}

// setProcAttr configures the command so the child process belongs to its own
// process group. This ensures SIGTERM reaches the entire subprocess tree.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// attachLifetime is a no-op on macOS: there is no Pdeathsig or Job Object
// equivalent. openvpn is stopped by the graceful shutdown path (SIGTERM via
// the management interface); a hard crash of Kongtrol can orphan it.
func attachLifetime(*process) error { return nil }

// releaseLifetime is a no-op on macOS.
func releaseLifetime(*process) {}
