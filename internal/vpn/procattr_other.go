//go:build !windows

package vpn

import "os/exec"

// HideChildWindow is a no-op on platforms with no console-window concept
// equivalent to Windows'. See procattr_windows.go.
func HideChildWindow(cmd *exec.Cmd) {}
