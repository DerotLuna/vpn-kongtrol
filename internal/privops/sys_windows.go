//go:build windows

package privops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	maxIfaceRunes    = 256
	ifaceAllowsSpace = true
	maxWGNameLen     = 32
	// privateDirSDDL: owner Administrators, protected DACL with full access
	// for SYSTEM and Administrators only, inherited by children.
	privateDirSDDL = "O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	// trustedInstallerSID owns most of %SystemRoot% and Program Files.
	trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1850883334-2271476464"
)

// ProgramDataDir returns %ProgramData%\Kongtrol.
func ProgramDataDir() string {
	base, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil || base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "Kongtrol")
}

// DefaultStateDir returns the privileged-only state directory.
func DefaultStateDir() string { return filepath.Join(ProgramDataDir(), "run") }

func programFiles() string {
	p, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil || p == "" {
		return `C:\Program Files`
	}
	return p
}

func systemDir() string {
	d, err := windows.GetSystemDirectory()
	if err != nil || d == "" {
		return `C:\Windows\System32`
	}
	return d
}

func isAdminOrSystem(s *windows.SID) bool {
	return s != nil && (s.IsWellKnown(windows.WinLocalSystemSid) || s.IsWellKnown(windows.WinBuiltinAdministratorsSid) ||
		s.String() == trustedInstallerSID)
}

// EnsurePrivateDir creates dir (and parents) and forces an Administrators
// owner with a protected SYSTEM+Administrators-only DACL. A pre-existing
// directory that is a link or is owned by anyone else (e.g. planted by a
// standard user under %ProgramData%, which users may write) is removed
// first so nothing a user created survives inside it.
func EnsurePrivateDir(dir string) error {
	if fi, err := os.Lstat(dir); err == nil {
		bad := !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0
		if !bad {
			sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
			if err != nil {
				return err
			}
			owner, _, err := sd.Owner()
			bad = err != nil || !isAdminOrSystem(owner)
		}
		if bad {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove untrusted %s: %w", dir, err)
			}
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString(privateDirSDDL)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		owner, nil, dacl, nil)
}

// ensurePrivateDir is a variable so tests can run unelevated.
var ensurePrivateDir = EnsurePrivateDir

func selfCaller() Caller {
	t := windows.GetCurrentProcessToken()
	c := Caller{UID: -1, Admin: t.IsElevated(), Name: "self"}
	if u, err := t.GetTokenUser(); err == nil {
		c.ID = u.User.Sid.String()
		if acct, dom, _, err := u.User.Sid.LookupAccount(""); err == nil {
			c.Name = dom + `\` + acct
		}
	}
	return c
}

// IsElevated reports whether this process runs as an elevated administrator.
func IsElevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

func safeEnv() []string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return []string{"SystemRoot=" + root, "PATH=" + systemDir() + ";" + root}
}

func run(ctx context.Context, exe string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = safeEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s", filepath.Base(exe), strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func netsh(ctx context.Context, args ...string) (string, error) {
	return run(ctx, filepath.Join(systemDir(), "netsh.exe"), args...)
}

func ipFamily(a netip.Addr) string {
	if a.Is4() {
		return "ipv4"
	}
	return "ipv6"
}

// ── Routes ──────────────────────────────────────────────────────────────

func sysAddRoute(ctx context.Context, r Route) error {
	args := []string{"interface", ipFamily(r.Destination.Addr()), "add", "route",
		"prefix=" + r.Destination.String(), "interface=" + r.Interface}
	if r.Gateway.IsValid() {
		args = append(args, "nexthop="+r.Gateway.String())
	}
	args = append(args, "metric="+strconv.Itoa(r.Metric), "store=active")
	_, err := netsh(ctx, args...)
	if err != nil && routeExists(err) {
		return nil
	}
	return err
}

func sysDeleteRoute(ctx context.Context, r Route) error {
	args := []string{"interface", ipFamily(r.Destination.Addr()), "delete", "route",
		"prefix=" + r.Destination.String(), "interface=" + r.Interface}
	if r.Gateway.IsValid() {
		args = append(args, "nexthop="+r.Gateway.String())
	}
	_, err := netsh(ctx, args...)
	return err
}

func routeExists(err error) bool {
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "already exists") || strings.Contains(m, "ya existe")
}

// ── DNS ─────────────────────────────────────────────────────────────────

// dnsSnapshot holds the static IPv4 resolvers configured before SetDNS
// (empty = DHCP-assigned), read from the TCP/IP registry key so DHCP
// servers are not mistaken for static ones.
type dnsSnapshot struct{ static []string }

func adapterGUID(name string) (string, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}
	size := uint32(15000)
	for i := 0; i < 3; i++ {
		buf := make([]byte, size)
		aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err = windows.GetAdaptersAddresses(windows.AF_UNSPEC, 0, 0, aa, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return "", err
		}
		for a := aa; a != nil; a = a.Next {
			if int(a.IfIndex) == ifi.Index {
				return windows.BytePtrToString(a.AdapterName), nil
			}
		}
		break
	}
	return "", fmt.Errorf("adapter for %q not found", name)
}

func sysSnapshotDNS(_ context.Context, iface string) (dnsSnapshot, error) {
	guid, err := adapterGUID(iface)
	if err != nil {
		return dnsSnapshot{}, err
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\`+guid, registry.QUERY_VALUE)
	if err != nil {
		return dnsSnapshot{}, nil // no key: treat as DHCP
	}
	defer k.Close()
	v, _, _ := k.GetStringValue("NameServer")
	var snap dnsSnapshot
	for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }) {
		if a, err := netip.ParseAddr(f); err == nil && a.Is4() {
			snap.static = append(snap.static, a.String())
		}
	}
	return snap, nil
}

// dnsListArgs builds the netsh calls that make servers iface's resolver
// list (empty = back to DHCP). Named parameters, per Microsoft's
// "netsh interface" reference:
//
//	set dnsservers [name=]<string> [source=]dhcp|static [address=]<IP>|none [register=]none|primary|both [validate=]yes|no
//	add dnsservers [name=]<string> [address=]<IP> [index=]<integer> [validate=]yes|no
//
// Named form keeps an interface name with spaces one argv element and can
// never be read as another parameter (names are validated against '=').
func dnsListArgs(fam, iface string, servers []string) [][]string {
	if len(servers) == 0 {
		return [][]string{{"interface", fam, "set", "dnsservers", "name=" + iface, "source=dhcp"}}
	}
	out := [][]string{{"interface", fam, "set", "dnsservers", "name=" + iface,
		"source=static", "address=" + servers[0], "register=primary", "validate=no"}}
	for i, s := range servers[1:] {
		out = append(out, []string{"interface", fam, "add", "dnsservers", "name=" + iface,
			"address=" + s, "index=" + strconv.Itoa(i+2), "validate=no"})
	}
	return out
}

func setDNSList(ctx context.Context, fam, iface string, servers []string) error {
	for _, args := range dnsListArgs(fam, iface, servers) {
		if _, err := netsh(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

func sysSetDNS(ctx context.Context, req DNSRequest) error {
	var v4, v6 []string
	for _, s := range req.Servers {
		if s.Is4() {
			v4 = append(v4, s.String())
		} else {
			v6 = append(v6, s.String())
		}
	}
	if len(v4) > 0 {
		if err := setDNSList(ctx, "ipv4", req.Interface, v4); err != nil {
			return err
		}
	}
	if len(v6) > 0 {
		return setDNSList(ctx, "ipv6", req.Interface, v6)
	}
	return nil
}

func sysRestoreDNS(ctx context.Context, iface string, snap dnsSnapshot) error {
	return setDNSList(ctx, "ipv4", iface, snap.static)
}

// ── Kill switch (Windows Firewall, same rule names as internal/security) ─

var ksRuleNames = []string{"KongtrolKillSwitchBlock", "KongtrolKillSwitchAllow",
	"KongtrolKillSwitchEndpoint", "KongtrolKillSwitchLAN", "KongtrolKillSwitchLoopback"}

func ksRule(name, action string, extra ...string) []string {
	return append([]string{"advfirewall", "firewall", "add", "rule", "name=" + name,
		"dir=out", "action=" + action, "enable=yes", "profile=any"}, extra...)
}

func ifaceIPv4s(name string) []string {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil
	}
	addrs, _ := ifi.Addrs()
	var out []string
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
			out = append(out, n.IP.String())
		}
	}
	return out
}

func sysKillSwitchEnable(ctx context.Context, req KillSwitchRequest) error {
	rules := [][]string{
		ksRule("KongtrolKillSwitchBlock", "block"),
		ksRule("KongtrolKillSwitchLoopback", "allow", "remoteip=127.0.0.0/8"),
	}
	var local []string
	for _, i := range req.Interfaces {
		local = append(local, ifaceIPv4s(i)...)
	}
	if len(local) > 0 {
		rules = append(rules, ksRule("KongtrolKillSwitchAllow", "allow", "localip="+strings.Join(local, ",")))
	}
	if len(req.Endpoints) > 0 {
		eps := make([]string, len(req.Endpoints))
		for i, e := range req.Endpoints {
			eps[i] = e.String()
		}
		rules = append(rules, ksRule("KongtrolKillSwitchEndpoint", "allow", "remoteip="+strings.Join(eps, ",")))
	}
	if req.AllowLAN {
		for _, s := range []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"} {
			rules = append(rules, ksRule("KongtrolKillSwitchLAN", "allow", "remoteip="+s))
		}
	}
	for _, r := range rules {
		if _, err := netsh(ctx, r...); err != nil {
			_ = sysKillSwitchDisable(ctx)
			return err
		}
	}
	return nil
}

func sysKillSwitchDisable(ctx context.Context) error {
	for _, n := range ksRuleNames {
		_, _ = netsh(ctx, "advfirewall", "firewall", "delete", "rule", "name="+n)
	}
	return nil
}

// ── WireGuard (wireguard.exe tunnel services) ───────────────────────────

func wgExe(name string) (string, error) {
	p := filepath.Join(programFiles(), "WireGuard", name)
	if fi, err := os.Stat(p); err != nil || !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s not installed", ErrNotFound, p)
	}
	return p, nil
}

func wgServiceName(name string) string { return "WireGuardTunnel$" + name }

func wgServiceState(name string) (svc.State, bool) {
	m, err := mgr.Connect()
	if err != nil {
		return 0, false
	}
	defer m.Disconnect()
	s, err := m.OpenService(wgServiceName(name))
	if err != nil {
		return 0, false
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return 0, true
	}
	return st.State, true
}

func sysWGUp(ctx context.Context, name, confPath string) error {
	if _, exists := wgServiceState(name); exists {
		_ = sysWGDown(ctx, name, confPath)
	}
	exe, err := wgExe("wireguard.exe")
	if err != nil {
		return err
	}
	if _, err := run(ctx, exe, "/installtunnelservice", confPath); err != nil {
		return err
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if st, ok := wgServiceState(name); ok && (st == svc.Running || st == svc.StartPending) {
			setWGManualStart(name)
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	st, _ := wgServiceState(name)
	return fmt.Errorf("WireGuard tunnel service did not start (state %d); see Event Viewer > Application", st)
}

func setWGManualStart(name string) {
	m, err := mgr.Connect()
	if err != nil {
		return
	}
	defer m.Disconnect()
	s, err := m.OpenService(wgServiceName(name))
	if err != nil {
		return
	}
	defer s.Close()
	if c, err := s.Config(); err == nil {
		c.StartType = mgr.StartManual
		_ = s.UpdateConfig(c)
	}
}

func sysWGDown(_ context.Context, name, _ string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	s, err := m.OpenService(wgServiceName(name))
	if err != nil {
		m.Disconnect()
		return nil // already gone
	}
	if st, err := s.Query(); err == nil && st.State != svc.Stopped && st.State != svc.StopPending {
		if _, err := s.Control(svc.Stop); err == nil {
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if st, err := s.Query(); err != nil || st.State == svc.Stopped {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
		}
	}
	derr := s.Delete()
	s.Close()
	m.Disconnect()
	if derr != nil {
		return derr
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := wgServiceState(name); !ok {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func sysWGSet(ctx context.Context, req WGAllowedIPsRequest) error {
	exe, err := wgExe("wg.exe")
	if err != nil {
		return err
	}
	ips := make([]string, len(req.AllowedIPs))
	for i, p := range req.AllowedIPs {
		ips[i] = p.String()
	}
	_, err = run(ctx, exe, "set", req.Interface, "peer", req.PeerPublicKey, "allowed-ips", strings.Join(ips, ","))
	return err
}

// ── OpenVPN ─────────────────────────────────────────────────────────────

// openvpnBinary resolves openvpn.exe from the installer's registry key or
// Program Files only, and requires it to live under Program Files (an
// administrator-only location). PATH and request data are never used.
func openvpnBinary() (string, error) {
	var cands []string
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\OpenVPN`, registry.QUERY_VALUE); err == nil {
		if p, _, err := k.GetStringValue("exe_path"); err == nil {
			cands = append(cands, p)
		}
		k.Close()
	}
	cands = append(cands, filepath.Join(programFiles(), "OpenVPN", "bin", "openvpn.exe"))
	pf := strings.ToLower(filepath.Clean(programFiles())) + `\`
	for _, p := range cands {
		p = filepath.Clean(p)
		real, err := filepath.EvalSymlinks(p)
		if err != nil || !strings.HasPrefix(strings.ToLower(real), pf) {
			continue
		}
		if fi, err := os.Stat(real); err == nil && fi.Mode().IsRegular() {
			return real, nil
		}
	}
	return "", errors.New("openvpn.exe not found under Program Files")
}

func configureOVPNCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
		HideWindow:    true,
	}
}

// signalOVPNStop: Windows has no SIGTERM; the caller signals openvpn via its
// management connection and the helper only waits, then kills.
func signalOVPNStop(*os.Process) {}

// ── Hosts ───────────────────────────────────────────────────────────────

func hostsPath() (string, error) {
	return filepath.Join(systemDir(), "drivers", "etc", "hosts"), nil
}

// ── Trusted file read ───────────────────────────────────────────────────

const writeMask = windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA |
	windows.FILE_WRITE_ATTRIBUTES | windows.WRITE_DAC | windows.WRITE_OWNER | windows.DELETE |
	windows.GENERIC_WRITE | windows.GENERIC_ALL

// readTrustedFile opens path while impersonating the caller (so the OS
// access check is the caller's, never the helper's), then requires: a
// regular non-link file no larger than max, owned by the caller, a trusted
// owner, Administrators or SYSTEM, and with no ACE granting write access to
// anyone else. It returns the content read from that single handle and the
// resolved path.
func readTrustedFile(c Caller, path string, max int64, trusted []string) ([]byte, string, error) {
	runtime.LockOSThread()
	if c.Token != 0 {
		if err := windows.SetThreadToken(nil, windows.Token(c.Token)); err != nil {
			runtime.UnlockOSThread()
			return nil, "", fmt.Errorf("%w: impersonate caller: %v", ErrDenied, err)
		}
	}
	data, resolved, err := readTrustedFileAsThread(c, path, max, trusted)
	if c.Token != 0 {
		if rerr := windows.RevertToSelf(); rerr != nil {
			// Keep the thread locked: it dies with this goroutine instead of
			// being reused while still impersonating.
			return nil, "", fmt.Errorf("%w: revert impersonation: %v", ErrFailed, rerr)
		}
	}
	runtime.UnlockOSThread()
	return data, resolved, err
}

func readTrustedFileAsThread(c Caller, path string, max int64, trusted []string) ([]byte, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s: %v", ErrNotFound, path, err)
	}
	p16, err := windows.UTF16PtrFromString(resolved)
	if err != nil {
		return nil, "", invalid("bad path")
	}
	h, err := windows.CreateFile(p16, windows.GENERIC_READ|windows.READ_CONTROL, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, "", fmt.Errorf("%w: open %s: %v", ErrDenied, resolved, err)
	}
	f := os.NewFile(uintptr(h), resolved)
	defer f.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrFailed, err)
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT) != 0 {
		return nil, "", invalid("%s is not a regular file", resolved)
	}
	if size := int64(info.FileSizeHigh)<<32 | int64(info.FileSizeLow); size > max {
		return nil, "", invalid("%s is larger than %d bytes", resolved, max)
	}
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return nil, "", fmt.Errorf("%w: read file security: %v", ErrFailed, err)
	}
	allowed := func(s *windows.SID) bool {
		if s == nil {
			return false
		}
		id := s.String()
		if isAdminOrSystem(s) || id == c.ID || s.IsWellKnown(windows.WinCreatorOwnerSid) || id == "S-1-3-4" {
			return true
		}
		for _, t := range trusted {
			if id == t {
				return true
			}
		}
		return false
	}
	owner, _, err := sd.Owner()
	if err != nil || !allowed(owner) {
		return nil, "", denied(DenyFileOwner, "%s must be owned by you or an administrator", resolved)
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return nil, "", denied(DenyFileWritable, "%s has no access list (anyone can change it)", resolved)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return nil, "", fmt.Errorf("%w: %v", ErrFailed, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Mask&writeMask != 0 && !allowed(sid) {
			return nil, "", denied(DenyFileWritable, "%s can be modified by other users (%s); move it to your user folder", resolved, sid.String())
		}
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read %s: %v", ErrFailed, resolved, err)
	}
	if int64(len(data)) > max {
		return nil, "", invalid("%s is larger than %d bytes", resolved, max)
	}
	return bytes.Clone(data), resolved, nil
}

// serviceExe returns the executable of a service command line
// (`"C:\Program Files\WireGuard\wireguard.exe" /tunnelservice "x.conf"`).
func serviceExe(cmdline string) string {
	cmdline = strings.TrimSpace(cmdline)
	if strings.HasPrefix(cmdline, `"`) {
		if end := strings.Index(cmdline[1:], `"`); end >= 0 {
			return cmdline[1 : end+1]
		}
		return strings.Trim(cmdline, `"`)
	}
	if i := strings.Index(strings.ToLower(cmdline), ".exe"); i >= 0 {
		return cmdline[:i+4]
	}
	if f := strings.Fields(cmdline); len(f) > 0 {
		return f[0]
	}
	return ""
}

// sysWGIsTunnel reports whether WireGuardTunnel$<name> exists, and denies
// (DenyNotWireGuard) a service of that name that does not run the
// admin-owned wireguard.exe.
func sysWGIsTunnel(_ context.Context, name string) (bool, error) {
	m, err := mgr.Connect()
	if err != nil {
		return false, err
	}
	defer m.Disconnect()
	s, err := m.OpenService(wgServiceName(name))
	if err != nil {
		return false, nil
	}
	defer s.Close()
	c, err := s.Config()
	if err != nil {
		return false, err
	}
	want, err := wgExe("wireguard.exe")
	if err != nil {
		return false, err
	}
	if got := serviceExe(c.BinaryPathName); !strings.EqualFold(filepath.Clean(got), filepath.Clean(want)) {
		return false, denied(DenyNotWireGuard, "service %s runs %q, not %s", wgServiceName(name), got, want)
	}
	return true, nil
}
