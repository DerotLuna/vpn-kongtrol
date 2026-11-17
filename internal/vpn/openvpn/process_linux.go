//go:build linux

package openvpn

import (
	"os/exec"
	"syscall"
)

// lifetimeHandle is unused on Linux: Pdeathsig needs no handle.
type lifetimeHandle = struct{}

// setProcAttr configures the command so the child process belongs to its own
// process group. This ensures SIGTERM reaches the entire subprocess tree.
//
// Pdeathsig makes the kernel send SIGTERM to openvpn when Kongtrol dies,
// even on a crash or SIGKILL, so the tunnel never outlives Kongtrol.
// Note: it fires when the creating OS thread exits, not the whole process;
// Go's os/exec forks from a thread that lives as long as the runtime keeps
// it, which is the accepted trade-off for this best-effort guard.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

// attachLifetime is a no-op on Linux: Pdeathsig (set in setProcAttr) already
// ties openvpn's life to Kongtrol's.
func attachLifetime(*process) error { return nil }

// releaseLifetime is a no-op on Linux.
func releaseLifetime(*process) {}
