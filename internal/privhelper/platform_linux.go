//go:build linux

package privhelper

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

const (
	unitName = "kongtrol-helper.service"
	unitPath = "/etc/systemd/system/" + unitName
)

func peerUID(c *net.UnixConn) (int, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var gerr error
	if err := raw.Control(func(fd uintptr) {
		var cred *unix.Ucred
		cred, gerr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if gerr == nil {
			uid = int(cred.Uid)
		}
	}); err != nil {
		return -1, err
	}
	return uid, gerr
}

func systemctl(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	exe := "/usr/bin/systemctl"
	if _, err := os.Stat(exe); err != nil {
		exe = "/bin/systemctl"
	}
	out, err := exec.CommandContext(ctx, exe, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %v: %w: %s", args, err, out)
	}
	return nil
}

func unitFile() string {
	return `[Unit]
Description=Kongtrol privileged helper (routes, DNS, firewall, VPN tunnels)
After=network.target

[Service]
Type=simple
ExecStart=` + BinaryPath() + ` helper run
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
`
}

func installService() error {
	if err := os.WriteFile(unitPath, []byte(unitFile()), 0o644); err != nil {
		return err
	}
	if err := chownRoot(unitPath, 0o644); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	if err := systemctl("enable", unitName); err != nil {
		return err
	}
	return systemctl("restart", unitName)
}

func stopService() { _ = systemctl("stop", unitName) }

func uninstallService() error {
	_ = systemctl("disable", "--now", unitName)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return systemctl("daemon-reload")
}

// QueryStatus works unprivileged.
func QueryStatus() State {
	if _, err := os.Stat(unitPath); err != nil {
		return State{}
	}
	return State{Installed: true, BinaryPath: BinaryPath(), Running: systemctl("is-active", "--quiet", unitName) == nil}
}
