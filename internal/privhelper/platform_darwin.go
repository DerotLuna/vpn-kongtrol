//go:build darwin

package privhelper

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	launchdLabel = "com.kongtrol.helper"
	plistPath    = "/Library/LaunchDaemons/" + launchdLabel + ".plist"
)

func peerUID(c *net.UnixConn) (int, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var gerr error
	if err := raw.Control(func(fd uintptr) {
		var cred *unix.Xucred
		cred, gerr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if gerr == nil {
			uid = int(cred.Uid)
		}
	}); err != nil {
		return -1, err
	}
	return uid, gerr
}

func launchctl(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/bin/launchctl", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("launchctl %v: %w: %s", args, err, out)
	}
	return string(out), nil
}

func plist() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>` + launchdLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + BinaryPath() + `</string>
		<string>helper</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><true/>
	<key>StandardErrorPath</key><string>/var/log/kongtrol-helper.err</string>
</dict>
</plist>
`
}

func installService() error {
	if err := os.WriteFile(plistPath, []byte(plist()), 0o644); err != nil {
		return err
	}
	if err := chownRoot(plistPath, 0o644); err != nil {
		return err
	}
	_, _ = launchctl("bootout", "system/"+launchdLabel)
	if _, err := launchctl("bootstrap", "system", plistPath); err != nil {
		return err
	}
	_, _ = launchctl("enable", "system/"+launchdLabel)
	_, err := launchctl("kickstart", "-k", "system/"+launchdLabel)
	return err
}

func stopService() { _, _ = launchctl("bootout", "system/"+launchdLabel) }

func uninstallService() error {
	stopService()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// QueryStatus works unprivileged.
func QueryStatus() State {
	if _, err := os.Stat(plistPath); err != nil {
		return State{}
	}
	out, err := launchctl("print", "system/"+launchdLabel)
	return State{Installed: true, BinaryPath: BinaryPath(), Running: err == nil && strings.Contains(out, "state = running")}
}
