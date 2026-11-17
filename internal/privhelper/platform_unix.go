//go:build linux || darwin

package privhelper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

const (
	binaryDir  = "/usr/local/libexec/kongtrol"
	configDir  = "/etc/kongtrol"
	auditPath  = "/var/log/kongtrol-helper.log"
	binaryName = "kongtrol-helper"
)

// BinaryPath is where `service install` copies the helper (root:root 0755).
func BinaryPath() string { return filepath.Join(binaryDir, binaryName) }

// ConfigPath is helper.json (root 0644).
func ConfigPath() string { return filepath.Join(configDir, "helper.json") }

// AuditPath is the helper audit log (root 0600).
func AuditPath() string { return auditPath }

func validateUserID(id string) error {
	if n, err := strconv.Atoi(id); err != nil || n < 0 || strconv.Itoa(n) != id {
		return fmt.Errorf("privhelper: invalid uid %q", id)
	}
	return nil
}

func rootOwnedPrivate(path string, mustBeRegular bool) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 || fi.Mode().Perm()&0o022 != 0 || (mustBeRegular && !fi.Mode().IsRegular()) {
		return fmt.Errorf("privhelper: %s must be owned by root and not writable by others", path)
	}
	return nil
}

func checkConfigFileTrusted(path string) error { return rootOwnedPrivate(path, true) }

func chownRoot(path string, mode os.FileMode) error {
	if err := os.Chown(path, 0, 0); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// listen creates the socket with a restrictive umask, then sets
// root:<installing user's group> 0660.
func listen(cfg Config) (net.Listener, error) {
	path := privops.SocketPath
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("privhelper: %s exists and is not a socket", path)
		}
		_ = os.Remove(path)
	}
	old := unix.Umask(0o177)
	l, err := net.Listen("unix", path)
	unix.Umask(old)
	if err != nil {
		return nil, err
	}
	l.(*net.UnixListener).SetUnlinkOnClose(true)
	if err := os.Chown(path, 0, cfg.GID); err != nil {
		_ = l.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0o660); err != nil {
		_ = l.Close()
		return nil, err
	}
	return l, nil
}

// authenticator verifies the peer uid from the kernel (SO_PEERCRED /
// LOCAL_PEERCRED): only the installing uid or root is served.
func authenticator(cfg Config) Authenticator {
	return func(conn net.Conn) (privops.Caller, func(), error) {
		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return privops.Caller{}, nil, errors.New("not a unix socket")
		}
		uid, err := peerUID(uc)
		if err != nil {
			return privops.Caller{}, nil, err
		}
		c := privops.Caller{ID: strconv.Itoa(uid), Name: strconv.Itoa(uid), UID: uid, Admin: uid == 0}
		if u, err := user.LookupId(c.ID); err == nil {
			c.Name = u.Username
		}
		if err := authorize(cfg, c); err != nil {
			return c, nil, fmt.Errorf("%w: uid %d", err, uid)
		}
		return c, nil, nil
	}
}

// Run is the `kongtrol helper run` entrypoint (launchd / systemd).
func Run(ctx context.Context) error {
	if !privops.IsElevated() {
		return errors.New("privhelper: run requires root")
	}
	return serve(ctx)
}

// Install copies the executable to /usr/local/libexec/kongtrol (plus any
// Homebrew tool copies in /usr/local/libexec/kongtrol/bin), writes
// /etc/kongtrol/helper.json, registers the system service and starts it.
// Must run as root.
func Install(o InstallOptions) error {
	if !privops.IsElevated() {
		return errors.New("privhelper: install requires root")
	}
	data, err := encodeConfig(Config{Format: ConfigFormat, UserID: o.UserID, UserName: o.UserName, GID: o.GID,
		Version: o.Version, InstalledAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	if o.GID < 0 {
		return errors.New("privhelper: invalid group id")
	}
	stopService()
	for _, d := range []string{binaryDir, configDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
		if err := chownRoot(d, 0o755); err != nil {
			return err
		}
		if !privops.TrustedDir(d) {
			return fmt.Errorf("privhelper: %s or a parent is not root-owned; fix its ownership and retry", d)
		}
	}
	if err := copyExecutable(o.SourceExe, BinaryPath()); err != nil {
		return err
	}
	// Homebrew/linuxbrew tools live in user-owned prefixes the helper must
	// not run from; copy them under binaryDir (removed by Uninstall).
	if err := installTools(); err != nil {
		return err
	}
	tmp := ConfigPath() + ".new"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := chownRoot(tmp, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, ConfigPath()); err != nil {
		return err
	}
	return installService()
}

// Uninstall stops and removes the service and its files.
func Uninstall() error {
	if !privops.IsElevated() {
		return errors.New("privhelper: uninstall requires root")
	}
	var errs []error
	if err := uninstallService(); err != nil {
		errs = append(errs, err)
	}
	for _, p := range []string{binaryDir, ConfigPath(), privops.SocketPath, privops.DefaultStateDir()} {
		if err := os.RemoveAll(p); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
