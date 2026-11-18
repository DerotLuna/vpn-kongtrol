package privops

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LocalOptions configures a Local.
type LocalOptions struct {
	// TrustedOwners are extra owner IDs (SID / uid) whose config files are
	// accepted besides the caller and administrators — the helper passes the
	// user recorded at install time.
	TrustedOwners []string
	// StateDir is a privileged-only directory for private copies of configs
	// (OpenVPN sessions, WireGuard tunnels). Default: DefaultStateDir().
	StateDir string
	// Now is the clock for lease expiry (tests). Default: time.Now.
	Now func() time.Time
}

// Local runs operations directly on this machine. The process must be
// elevated (administrator / root) for the system calls to succeed. Keep
// one Local per process: it tracks added routes, DNS snapshots and
// OpenVPN sessions in memory.
type Local struct {
	opts     LocalOptions
	mu       sync.Mutex
	routes   map[string][]Route // per interface, added by this Local
	dnsPrev  map[string]dnsSnapshot
	sessions map[string]*ovpnSession
	leases   *leaseTable
	// releaseFn undoes one resource of an expired lease (stubbed in tests).
	releaseFn func(context.Context, resource) error
}

// NewLocal returns a Local.
func NewLocal(opts LocalOptions) *Local {
	if opts.StateDir == "" {
		opts.StateDir = DefaultStateDir()
	}
	l := &Local{
		opts:     opts,
		routes:   map[string][]Route{},
		dnsPrev:  map[string]dnsSnapshot{},
		sessions: map[string]*ovpnSession{},
		leases:   newLeaseTable(opts.Now, LeaseTTL),
	}
	l.releaseFn = l.releaseResource
	return l
}

// interfaceExists is a variable so tests can stub it.
var interfaceExists = func(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

func requireIface(name string) error {
	if err := ValidateInterfaceName(name); err != nil {
		return err
	}
	if !interfaceExists(name) {
		return fmt.Errorf("%w: interface %q does not exist", ErrNotFound, name)
	}
	return nil
}

func failed(op string, err error) error {
	if err == nil {
		return nil
	}
	for _, s := range []error{ErrInvalidRequest, ErrDenied, ErrNotFound, ErrUnsupported} {
		if errors.Is(err, s) {
			return err
		}
	}
	return fmt.Errorf("%w: %s: %v", ErrFailed, op, err)
}

func (l *Local) Ping(ctx context.Context) (PingResult, error) {
	c := CallerFrom(ctx)
	return PingResult{Version: BuildVersion, Protocol: ProtocolVersion, Caller: c.Name}, nil
}

// ── Routes ──────────────────────────────────────────────────────────────

func (l *Local) AddRoute(ctx context.Context, r Route) error {
	r, err := NormalizeRoute(r)
	if err != nil {
		return err
	}
	if err := requireIface(r.Interface); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := sysAddRoute(ctx, r); err != nil {
		return failed("add route", err)
	}
	for _, cur := range l.routes[r.Interface] {
		if cur.Destination == r.Destination {
			return nil
		}
	}
	l.routes[r.Interface] = append(l.routes[r.Interface], r)
	return nil
}

func (l *Local) DeleteRoute(ctx context.Context, r Route) error {
	r, err := NormalizeRoute(r)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := sysDeleteRoute(ctx, r); err != nil {
		return failed("delete route", err)
	}
	l.untrack(r)
	return nil
}

func (l *Local) untrack(r Route) {
	cur := l.routes[r.Interface]
	out := cur[:0]
	for _, c := range cur {
		if c.Destination != r.Destination {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		delete(l.routes, r.Interface)
	} else {
		l.routes[r.Interface] = out
	}
}

// FlushRoutes deletes only routes this Local added on iface (best effort).
func (l *Local) FlushRoutes(ctx context.Context, iface string) error {
	if err := ValidateInterfaceName(iface); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.routes[iface] {
		_ = sysDeleteRoute(ctx, r)
	}
	delete(l.routes, iface)
	return nil
}

// ── DNS ─────────────────────────────────────────────────────────────────

func (l *Local) SetDNS(ctx context.Context, req DNSRequest) error {
	req, err := NormalizeDNS(req)
	if err != nil {
		return err
	}
	if err := requireIface(req.Interface); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.dnsPrev[req.Interface]; !ok {
		snap, err := sysSnapshotDNS(ctx, req.Interface)
		if err != nil {
			return failed("read current DNS", err)
		}
		l.dnsPrev[req.Interface] = snap
	}
	if err := sysSetDNS(ctx, req); err != nil {
		return failed("set DNS", err)
	}
	l.leases.own(l.leaseOf(ctx), resource{resDNS, req.Interface})
	return nil
}

func (l *Local) RestoreDNS(ctx context.Context, iface string) error {
	if err := ValidateInterfaceName(iface); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.leases.release(resource{resDNS, iface})
	snap, ok := l.dnsPrev[iface]
	if !ok {
		return nil // nothing applied by us
	}
	delete(l.dnsPrev, iface)
	return failed("restore DNS", sysRestoreDNS(ctx, iface, snap))
}

// ── Kill switch ─────────────────────────────────────────────────────────

func (l *Local) KillSwitchEnable(ctx context.Context, req KillSwitchRequest) error {
	req, err := NormalizeKillSwitch(req)
	if err != nil {
		return err
	}
	for _, i := range req.Interfaces {
		if err := requireIface(i); err != nil {
			return err
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = sysKillSwitchDisable(ctx) // drop stale rules first
	if err := sysKillSwitchEnable(ctx, req); err != nil {
		l.leases.release(resource{resKillSwitch, ""})
		return failed("enable kill switch", err)
	}
	l.leases.own(l.leaseOf(ctx), resource{resKillSwitch, ""})
	return nil
}

func (l *Local) KillSwitchDisable(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.leases.release(resource{resKillSwitch, ""})
	return failed("disable kill switch", sysKillSwitchDisable(ctx))
}

// ── WireGuard ───────────────────────────────────────────────────────────

func (l *Local) wgDir() string { return filepath.Join(l.opts.StateDir, "wg") }

// wgManaged reports whether name was installed through privops (its
// private copy exists) or is a stale tunnel from older Kongtrol builds.
func (l *Local) wgManaged(name string) bool {
	if strings.HasPrefix(name, "kongtrol-wg-") {
		return true
	}
	fi, err := os.Lstat(filepath.Join(l.wgDir(), name+".conf"))
	return err == nil && fi.Mode().IsRegular()
}

func (l *Local) WGInstallTunnel(ctx context.Context, req WGInstallRequest) (WGTunnel, error) {
	if err := ValidateConfigPath(req.ConfigPath, ".conf"); err != nil {
		return WGTunnel{}, err
	}
	name := req.Name
	if name == "" {
		base := filepath.Base(req.ConfigPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if err := ValidateWGName(name); err != nil {
		return WGTunnel{}, err
	}
	data, _, err := readTrustedFile(CallerFrom(ctx), req.ConfigPath, MaxConfigSize, l.opts.TrustedOwners)
	if err != nil {
		return WGTunnel{}, err
	}
	if err := ValidateWireGuardConfig(data); err != nil {
		return WGTunnel{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := ensurePrivateDir(l.wgDir()); err != nil {
		return WGTunnel{}, failed("state dir", err)
	}
	copyPath := filepath.Join(l.wgDir(), name+".conf")
	if l.wgManaged(name) {
		_ = sysWGDown(ctx, name, copyPath)
	}
	if err := writePrivateFile(copyPath, data); err != nil {
		return WGTunnel{}, failed("write tunnel config", err)
	}
	if err := sysWGUp(ctx, name, copyPath); err != nil {
		_ = os.Remove(copyPath)
		return WGTunnel{}, failed("start WireGuard tunnel", err)
	}
	return WGTunnel{Name: name}, nil
}

// System hooks for WireGuard tunnels (stubbed in tests).
var (
	wgIsTunnelFn = sysWGIsTunnel
	wgDownFn     = sysWGDown
	wgSetFn      = sysWGSet
)

// wgAuthorize allows managing tunnel name: it was started through privops,
// or the caller adopts it with its own profile config. Adoption covers
// tunnels brought up by older Kongtrol builds (UAC /installtunnelservice on
// the profile file or a temp copy of it) and by the WireGuard app: the
// tunnel must be a genuine WireGuard tunnel (Windows: service
// WireGuardTunnel$<name> running the admin-owned wireguard.exe; Linux: a
// wireguard link; macOS: a wireguard-go tunnel), configPath must pass the
// same checks as WGInstallTunnel, and its base name must equal name. The
// validated config becomes the private copy, so the tunnel is managed from
// then on (only with adopt; WGStats passes false and writes nothing).
// present is false when no such tunnel exists (nothing to manage).
// Call with l.mu held.
func (l *Local) wgAuthorize(ctx context.Context, name, configPath string, adopt bool) (present bool, err error) {
	if l.wgManaged(name) {
		return true, nil
	}
	if configPath == "" {
		return false, denied(DenyWGNotManaged, "tunnel %q was not started by Kongtrol and no profile config was given", name)
	}
	if err := ValidateConfigPath(configPath, ".conf"); err != nil {
		return false, err
	}
	base := filepath.Base(configPath)
	if !strings.EqualFold(strings.TrimSuffix(base, filepath.Ext(base)), name) {
		return false, denied(DenyWGNotManaged, "tunnel %q does not match profile config %s", name, base)
	}
	data, _, err := readTrustedFile(CallerFrom(ctx), configPath, MaxConfigSize, l.opts.TrustedOwners)
	if err != nil {
		return false, err
	}
	if err := ValidateWireGuardConfig(data); err != nil {
		return false, err
	}
	ok, err := wgIsTunnelFn(ctx, name)
	if err != nil {
		return false, failed("inspect WireGuard tunnel", err)
	}
	if !ok {
		return false, nil
	}
	if !adopt {
		return true, nil // read-only caller (WGStats): accept, write nothing
	}
	if err := ensurePrivateDir(l.wgDir()); err != nil {
		return false, failed("state dir", err)
	}
	if err := writePrivateFile(filepath.Join(l.wgDir(), name+".conf"), data); err != nil {
		return false, failed("adopt tunnel config", err)
	}
	return true, nil
}

func (l *Local) WGRemoveTunnel(ctx context.Context, req WGRemoveRequest) error {
	if err := ValidateWGName(req.Name); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	present, err := l.wgAuthorize(ctx, req.Name, req.ConfigPath, true)
	if err != nil {
		return err
	}
	if !present {
		return nil // no such tunnel: already down
	}
	copyPath := filepath.Join(l.wgDir(), req.Name+".conf")
	if err := wgDownFn(ctx, req.Name, copyPath); err != nil {
		return failed("stop WireGuard tunnel", err)
	}
	_ = os.Remove(copyPath)
	return nil
}

func (l *Local) WGSetAllowedIPs(ctx context.Context, req WGAllowedIPsRequest) error {
	req, err := NormalizeAllowedIPs(req)
	if err != nil {
		return err
	}
	l.mu.Lock()
	present, err := l.wgAuthorize(ctx, req.Interface, req.ConfigPath, true)
	l.mu.Unlock()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("%w: WireGuard tunnel %q is not running", ErrNotFound, req.Interface)
	}
	return failed("set allowed IPs", wgSetFn(ctx, req))
}

// wgDumpFn reads `wg show <name> dump` (stubbed in tests).
var wgDumpFn = sysWGDump

// WGStats reads the per-peer handshake and transfer counters of a tunnel
// Kongtrol manages (or the caller's own profile tunnel, without adopting
// it). Only peer public keys leave the helper: the interface private key
// and preshared keys in the dump are dropped by ParseWGDump.
func (l *Local) WGStats(ctx context.Context, req WGStatsRequest) (WGStats, error) {
	if err := ValidateWGName(req.Interface); err != nil {
		return WGStats{}, err
	}
	l.mu.Lock()
	present, err := l.wgAuthorize(ctx, req.Interface, req.ConfigPath, false)
	l.mu.Unlock()
	if err != nil {
		return WGStats{}, err
	}
	if !present {
		return WGStats{}, fmt.Errorf("%w: WireGuard tunnel %q is not running", ErrNotFound, req.Interface)
	}
	out, err := wgDumpFn(ctx, req.Interface)
	if err != nil {
		// The command's output may echo the dump: never return it.
		return WGStats{}, fmt.Errorf("%w: read WireGuard tunnel state", ErrFailed)
	}
	return ParseWGDump(out)
}

// ── OpenVPN ─────────────────────────────────────────────────────────────

const ovpnLogLines = 200

// ovpnPushLines bounds the PUSH_REPLY lines kept per session.
const ovpnPushLines = 8

type ovpnSession struct {
	id, owner string
	dir       string
	cmd       *exec.Cmd
	exited    chan struct{}
	exitErr   error
	logMu     sync.Mutex
	log       []string
	push      []string // latest PUSH_REPLY lines (OpenVPNState.PushReply)
}

func (s *ovpnSession) addLog(line string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	if len(s.log) >= ovpnLogLines {
		s.log = s.log[1:]
	}
	s.log = append(s.log, line)
	if strings.Contains(line, "PUSH_REPLY") {
		s.push = lastN(append(s.push, line), ovpnPushLines)
	}
}

func (s *ovpnSession) pushReply() []string {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	return append([]string(nil), s.push...)
}

func (s *ovpnSession) tail() []string {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	return append([]string(nil), s.log...)
}

func (s *ovpnSession) running() bool {
	select {
	case <-s.exited:
		return false
	default:
		return true
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (l *Local) OpenVPNStart(ctx context.Context, req OpenVPNStartRequest) (OpenVPNSession, error) {
	caller := CallerFrom(ctx)
	if err := ValidateConfigPath(req.ConfigPath, ".ovpn", ".conf"); err != nil {
		return OpenVPNSession{}, err
	}
	if (req.CertPath == "") != (req.KeyPath == "") {
		return OpenVPNSession{}, invalid("cert and key must be given together")
	}
	for _, p := range []string{req.CertPath, req.KeyPath} {
		if p != "" {
			if err := ValidateConfigPath(p, ".crt", ".pem", ".key", ".cer"); err != nil {
				return OpenVPNSession{}, err
			}
		}
	}
	data, resolved, err := readTrustedFile(caller, req.ConfigPath, MaxConfigSize, l.opts.TrustedOwners)
	if err != nil {
		return OpenVPNSession{}, err
	}
	baseDir := filepath.Dir(resolved)
	refs := func(p string) ([]byte, error) {
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		b, _, err := readTrustedFile(caller, filepath.Clean(p), MaxReferenceSize, l.opts.TrustedOwners)
		return b, err
	}
	conf, err := SanitizeOpenVPNConfig(data, refs)
	if err != nil {
		return OpenVPNSession{}, err
	}
	if req.CertPath != "" {
		for _, kv := range [][2]string{{"cert", req.CertPath}, {"key", req.KeyPath}} {
			b, err := refs(kv[1])
			if err != nil {
				return OpenVPNSession{}, err
			}
			blk, err := inlineBlock(kv[0], b, nil)
			if err != nil {
				return OpenVPNSession{}, err
			}
			conf = append(conf, blk...)
		}
	}
	bin, err := openvpnBinary()
	if err != nil {
		return OpenVPNSession{}, fmt.Errorf("%w: openvpn: %v", ErrNotFound, err)
	}

	id := randomHex(16)
	dir := filepath.Join(l.opts.StateDir, "openvpn", id)
	if err := ensurePrivateDir(filepath.Dir(dir)); err != nil {
		return OpenVPNSession{}, failed("state dir", err)
	}
	if err := ensurePrivateDir(dir); err != nil {
		return OpenVPNSession{}, failed("state dir", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	confPath := filepath.Join(dir, "config.ovpn")
	pwPath := filepath.Join(dir, "mgmt.pw")
	password := randomHex(16)
	if err := writePrivateFile(confPath, conf); err != nil {
		cleanup()
		return OpenVPNSession{}, failed("write config", err)
	}
	if err := writePrivateFile(pwPath, []byte(password+"\n")); err != nil {
		cleanup()
		return OpenVPNSession{}, failed("write config", err)
	}
	port, err := freePort()
	if err != nil {
		cleanup()
		return OpenVPNSession{}, failed("management port", err)
	}
	args := []string{
		"--cd", dir,
		"--config", confPath,
		// After --config so it overrides anything the file could say.
		"--script-security", "1",
		"--management", "127.0.0.1", strconv.Itoa(port), pwPath,
		"--management-query-passwords",
		"--verb", "3",
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = safeEnv()
	configureOVPNCmd(cmd)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	s := &ovpnSession{id: id, owner: caller.ID, dir: dir, cmd: cmd, exited: make(chan struct{})}
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		cleanup()
		return OpenVPNSession{}, failed("start openvpn", err)
	}
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			s.addLog(sc.Text())
		}
		_, _ = io.Copy(io.Discard, pr)
	}()
	go func() {
		s.exitErr = cmd.Wait()
		_ = pw.Close()
		close(s.exited)
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if err := waitForTCP(ctx, addr, s.exited, 20*time.Second); err != nil {
		_ = cmd.Process.Kill()
		<-s.exited
		cleanup()
		return OpenVPNSession{}, fmt.Errorf("%w: openvpn did not start: %v: %s", ErrFailed, err, strings.Join(lastN(s.tail(), 5), " | "))
	}
	l.mu.Lock()
	l.sessions[id] = s
	l.mu.Unlock()
	l.leases.own(l.leaseOf(ctx), resource{resOpenVPN, id})
	return OpenVPNSession{ID: id, PID: cmd.Process.Pid, MgmtAddr: addr, MgmtPassword: password}, nil
}

func (l *Local) session(ctx context.Context, id string) (*ovpnSession, error) {
	if err := ValidateSessionID(id); err != nil {
		return nil, err
	}
	l.mu.Lock()
	s, ok := l.sessions[id]
	l.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w: OpenVPN session", ErrNotFound)
	}
	c := CallerFrom(ctx)
	if s.owner != c.ID && !c.Admin {
		return nil, denied(DenySessionOwner, "session belongs to another user")
	}
	return s, nil
}

// OpenVPNStop ends a session. Callers should first send "signal SIGTERM"
// over their own management connection (OpenVPN accepts one management
// client at a time) so openvpn removes its routes; the helper then waits
// up to 10s for the exit before terminating the process.
func (l *Local) OpenVPNStop(ctx context.Context, id string) error {
	s, err := l.session(ctx, id)
	if err != nil {
		return err
	}
	l.stopSession(s)
	return nil
}

// stopSession ends s (waiting up to 10s for openvpn to exit cleanly) and
// forgets it.
func (l *Local) stopSession(s *ovpnSession) {
	l.leases.release(resource{resOpenVPN, s.id})
	if s.running() {
		signalOVPNStop(s.cmd.Process)
		select {
		case <-s.exited:
		case <-time.After(10 * time.Second):
			_ = s.cmd.Process.Kill()
			<-s.exited
		}
	}
	l.mu.Lock()
	delete(l.sessions, s.id)
	l.mu.Unlock()
	_ = os.RemoveAll(s.dir)
}

func (l *Local) OpenVPNStatus(ctx context.Context, id string) (OpenVPNState, error) {
	s, err := l.session(ctx, id)
	if err != nil {
		return OpenVPNState{}, err
	}
	st := OpenVPNState{Running: s.running(), LogTail: lastN(s.tail(), 50), PushReply: s.pushReply()}
	if !st.Running && s.exitErr != nil {
		st.ExitError = s.exitErr.Error()
	}
	return st, nil
}

// StopAll terminates every OpenVPN session (helper shutdown).
func (l *Local) StopAll() {
	l.mu.Lock()
	var all []*ovpnSession
	for _, s := range l.sessions {
		all = append(all, s)
	}
	l.mu.Unlock()
	for _, s := range all {
		if s.running() {
			_ = s.cmd.Process.Kill()
			<-s.exited
		}
		_ = os.RemoveAll(s.dir)
	}
}

func lastN(s []string, n int) []string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func waitForTCP(ctx context.Context, addr string, exited <-chan struct{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			return errors.New("openvpn exited")
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			_ = c.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("timeout waiting for the management interface")
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// ── Hosts ───────────────────────────────────────────────────────────────

func (l *Local) WriteHostsBlock(ctx context.Context, entries []HostEntry) error {
	entries, err := NormalizeHostEntries(entries)
	if err != nil {
		return err
	}
	path, err := hostsPath()
	if err != nil {
		return failed("hosts", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fi, err := os.Lstat(path)
	if err != nil {
		return failed("hosts", err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: hosts file is not a regular file", ErrFailed)
	}
	cur, err := os.ReadFile(path)
	if err != nil {
		return failed("hosts", err)
	}
	next, err := ReplaceHostsBlock(string(cur), entries)
	if err != nil {
		return failed("hosts", err)
	}
	if next != string(cur) {
		if err := os.WriteFile(path, []byte(next), fi.Mode().Perm()); err != nil {
			return failed("hosts", err)
		}
	}
	if len(entries) == 0 {
		l.leases.release(resource{resHosts, ""})
	} else {
		l.leases.own(l.leaseOf(ctx), resource{resHosts, ""})
	}
	return nil
}

// writePrivateFile writes data readable only by the privileged user.
func writePrivateFile(path string, data []byte) error {
	_ = os.Remove(path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

var _ Ops = (*Local)(nil)
