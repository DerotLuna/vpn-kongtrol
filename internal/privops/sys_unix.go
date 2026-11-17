//go:build linux || darwin

package privops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	maxIfaceRunes    = 15 // IFNAMSIZ-1
	ifaceAllowsSpace = false
	maxWGNameLen     = 15
)

// DefaultStateDir returns the root-only state directory.
func DefaultStateDir() string { return filepath.Join(runDir, "kongtrol-helper") }

// EnsurePrivateDir creates dir as root:root 0700. A pre-existing entry that
// is not a root-owned private directory is removed first.
func EnsurePrivateDir(dir string) error {
	if fi, err := os.Lstat(dir); err == nil {
		st, _ := fi.Sys().(*syscall.Stat_t)
		if !fi.IsDir() || st == nil || st.Uid != 0 || fi.Mode().Perm()&0o077 != 0 {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		if err := os.Chown(dir, 0, 0); err != nil {
			return err
		}
	}
	return os.Chmod(dir, 0o700)
}

// ensurePrivateDir is a variable so tests can run unelevated.
var ensurePrivateDir = EnsurePrivateDir

func selfCaller() Caller {
	uid := os.Geteuid()
	name := strconv.Itoa(uid)
	if u, err := user.LookupId(name); err == nil {
		name = u.Username
	}
	return Caller{ID: strconv.Itoa(uid), Name: name, Admin: uid == 0, UID: uid}
}

// IsElevated reports whether this process runs as root.
func IsElevated() bool { return os.Geteuid() == 0 }

// TrustedDir reports whether dir and all its parents are root-owned and
// not writable by group/others, so nothing in it can be swapped by a user.
func TrustedDir(dir string) bool {
	for d := filepath.Clean(dir); ; d = filepath.Dir(d) {
		fi, err := os.Stat(d)
		if err != nil {
			return false
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 || (fi.Mode().Perm()&0o022 != 0 && fi.Mode()&os.ModeSticky == 0) {
			return false
		}
		if d == "/" {
			return true
		}
	}
}

// trustedBinary returns the first candidate that resolves to a root-owned
// executable not writable by others inside a trusted directory chain.
func trustedBinary(cands ...string) (string, error) {
	for _, c := range cands {
		real, err := filepath.EvalSymlinks(c)
		if err != nil {
			continue
		}
		fi, err := os.Stat(real)
		if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o022 != 0 {
			continue
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); !ok || st.Uid != 0 {
			continue
		}
		if TrustedDir(filepath.Dir(real)) {
			return real, nil
		}
	}
	return "", fmt.Errorf("none of %s is installed in a root-owned location", strings.Join(cands, ", "))
}

// TrustedBinary is trustedBinary for `service install`, which uses it to
// prefer a root-owned system tool over copying a Homebrew one.
func TrustedBinary(cands ...string) (string, error) { return trustedBinary(cands...) }

// ToolsDir holds root-owned copies of tools found in user-owned prefixes
// (Homebrew, linuxbrew), made by `service install`. The helper tries it
// after the system locations.
const ToolsDir = "/usr/local/libexec/kongtrol/bin"

func toolPath(name string) string { return filepath.Join(ToolsDir, name) }

// safeEnv gives child tools a PATH made only of trusted directories, so a
// root-run script (wg-quick) can't pick up a user-writable binary.
func safeEnv() []string {
	var dirs []string
	// ToolsDir first: wg-quick's "env bash", wg and wireguard-go must find
	// the root-owned copies made at install time before macOS's bash 3.2.
	if TrustedDir(ToolsDir) {
		dirs = append(dirs, ToolsDir)
	}
	for _, d := range []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"} {
		if TrustedDir(d) {
			dirs = append(dirs, d)
		}
	}
	return []string{"PATH=" + strings.Join(dirs, ":"), "LANG=C", "LC_ALL=C"}
}

func run(ctx context.Context, exe string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = safeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s", filepath.Base(exe), strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ── WireGuard (wg-quick) ────────────────────────────────────────────────

func wgQuick() (string, error) {
	return trustedBinary("/usr/bin/wg-quick", "/usr/local/bin/wg-quick", "/opt/homebrew/bin/wg-quick", toolPath("wg-quick"))
}

func sysWGUp(ctx context.Context, _ string, confPath string) error {
	exe, err := wgQuick()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	_, err = run(ctx, exe, "up", confPath)
	return err
}

func sysWGDown(ctx context.Context, _ string, confPath string) error {
	exe, err := wgQuick()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	_, err = run(ctx, exe, "down", confPath)
	return err
}

func sysWGSet(ctx context.Context, req WGAllowedIPsRequest) error {
	exe, err := trustedBinary("/usr/bin/wg", "/usr/local/bin/wg", "/opt/homebrew/bin/wg", toolPath("wg"))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	ips := make([]string, len(req.AllowedIPs))
	for i, p := range req.AllowedIPs {
		ips[i] = p.String()
	}
	_, err = run(ctx, exe, "set", req.Interface, "peer", req.PeerPublicKey, "allowed-ips", strings.Join(ips, ","))
	return err
}

// ── OpenVPN ─────────────────────────────────────────────────────────────

func openvpnBinary() (string, error) {
	return trustedBinary("/usr/sbin/openvpn", "/usr/local/sbin/openvpn", "/usr/bin/openvpn",
		"/opt/local/sbin/openvpn", "/opt/homebrew/sbin/openvpn", toolPath("openvpn"))
}

func signalOVPNStop(p *os.Process) { _ = p.Signal(syscall.SIGTERM) }

// ── Hosts ───────────────────────────────────────────────────────────────

func hostsPath() (string, error) { return "/etc/hosts", nil }

// ── Trusted file read ───────────────────────────────────────────────────

// readTrustedFile resolves symlinks, opens the file once with O_NOFOLLOW
// and validates that same descriptor: regular file, at most max bytes, not
// writable by group/others, and owned by the caller or a trusted user — or
// by root and world-readable (so the helper never reveals a root-only file
// the caller could not read).
func readTrustedFile(c Caller, path string, max int64, trusted []string) ([]byte, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s: %v", ErrNotFound, path, err)
	}
	fd, err := unix.Open(resolved, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, "", fmt.Errorf("%w: open %s: %v", ErrDenied, resolved, err)
	}
	f := os.NewFile(uintptr(fd), resolved)
	defer f.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrFailed, err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, "", invalid("%s is not a regular file", resolved)
	}
	if st.Size > max {
		return nil, "", invalid("%s is larger than %d bytes", resolved, max)
	}
	if st.Mode&0o022 != 0 {
		return nil, "", denied(DenyFileWritable, "%s can be modified by other users; run: chmod go-w %s", resolved, resolved)
	}
	owner := strconv.FormatUint(uint64(st.Uid), 10)
	ok := owner == c.ID
	for _, t := range trusted {
		ok = ok || owner == t
	}
	if st.Uid == 0 {
		ok = c.UID == 0 || st.Mode&0o004 != 0
	}
	if !ok {
		return nil, "", denied(DenyFileOwner, "%s must be owned by you", resolved)
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read %s: %v", ErrFailed, resolved, err)
	}
	if int64(len(data)) > max {
		return nil, "", invalid("%s is larger than %d bytes", resolved, max)
	}
	return data, resolved, nil
}

var errNoDarwinService = errors.New("no network service for interface")

// rootOwnedFile reports whether path (not followed if a link) is owned by
// root and is a socket (socket=true) or a regular file.
func rootOwnedFile(path string, socket bool) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return false
	}
	if socket {
		return fi.Mode()&os.ModeSocket != 0
	}
	return fi.Mode().IsRegular()
}
