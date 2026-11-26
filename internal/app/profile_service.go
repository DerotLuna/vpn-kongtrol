package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

type Watchdog interface {
	MarkIntended(name string)
	MarkActive(name string)
}

type PolicyResolver interface {
	RegisterProfile(name, ifaceName, configPath string) error
	RegisterTunnel(name, iface string, dns []net.IP) error
	UnregisterProfile(name string)
}

type DNSManager interface {
	OnConnect(profile string, iface string, dnsServers []net.IP)
	OnDisconnect(profile string)
}

type ProfileService struct {
	cfg             *config.Config
	adapters        *vpn.Set
	watchdog        Watchdog
	policyResolver  PolicyResolver
	dnsMgr          DNSManager
	applyKill       func() error
	beginDisconnect func(string) (func(), error)
	format          func(key string, args ...any) string
	translate       func(key string) string
	emitAlert       func(level, profile, message string)
	logAudit        func(level, action, profile, message string)
	emitWarn        func(message string)
	emitStderr      func(message string)
	onAttach        func(name string, st vpn.AttachState)
	onIntent        func(name string, intendedDown bool)
	externalTunnels func() []ExternalTunnel

	// inflight tracks running Connect calls per profile so Disconnect can
	// cancel them (and wait for them) before it reports success.
	mu       sync.Mutex
	inflight map[string]map[uint64]*pendingConnect
	nextID   uint64
}

// pendingConnect is one running Connect: cancel stops it, done is closed
// when it has returned.
type pendingConnect struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// disconnectCancelWait bounds how long Disconnect waits for a cancelled
// Connect to return (an adapter step that ignores its context).
var disconnectCancelWait = 15 * time.Second

type ProfileServiceDeps struct {
	Cfg             *config.Config
	Adapters        *vpn.Set
	Watchdog        Watchdog
	PolicyResolver  PolicyResolver
	DNSManager      DNSManager
	ApplyKill       func() error
	BeginDisconnect func(string) (func(), error)
	Format          func(key string, args ...any) string
	Translate       func(key string) string
	EmitAlert       func(level, profile, message string)
	LogAudit        func(level, action, profile, message string)
	EmitWarn        func(message string)
	EmitStderr      func(message string)
	// OnAttach is called on every provider attach-phase change (optional).
	OnAttach func(name string, st vpn.AttachState)
	// OnIntent records that the user turned a tunnel on (false) or off
	// (true): the kill switch never blocks a tunnel the user turned off.
	OnIntent func(name string, intendedDown bool)
	// ExternalTunnels lists provider app tunnels up outside Kongtrol
	// (optional): a full-tunnel connect is refused while one is up.
	ExternalTunnels func() []ExternalTunnel
}

func NewProfileService(deps ProfileServiceDeps) *ProfileService {
	return &ProfileService{
		cfg:             deps.Cfg,
		adapters:        deps.Adapters,
		watchdog:        deps.Watchdog,
		policyResolver:  deps.PolicyResolver,
		dnsMgr:          deps.DNSManager,
		applyKill:       deps.ApplyKill,
		beginDisconnect: deps.BeginDisconnect,
		format:          deps.Format,
		translate:       deps.Translate,
		emitAlert:       deps.EmitAlert,
		logAudit:        deps.LogAudit,
		emitWarn:        deps.EmitWarn,
		emitStderr:      deps.EmitStderr,
		onAttach:        deps.OnAttach,
		onIntent:        deps.OnIntent,
		externalTunnels: deps.ExternalTunnels,
	}
}

func (s *ProfileService) Connect(ctx context.Context, name string) error {
	return s.connect(ctx, name, false)
}

// ErrTunnelNotAttached indicates that an attach-only startup found no tunnel.
var ErrTunnelNotAttached = errors.New("provider tunnel is not attached")

// ConnectAttached applies the connected profile lifecycle without starting an adapter.
func (s *ProfileService) ConnectAttached(ctx context.Context, name string) error {
	return s.connect(ctx, name, true)
}

func (s *ProfileService) connect(ctx context.Context, name string, attachOnly bool) error {
	adapter, ok := s.adapters.Get(name)
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.unknown_profile", name))
	}

	vpnCfg, ok := s.cfg.VPNs[name]
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.no_config_for_profile", name))
	}
	if adapter.Status().Normalize() != vpn.StatusConnected {
		// Never start a second tunnel that takes all traffic (ADR-0099).
		if err := s.CheckFullTunnelConflict(name); err != nil {
			s.audit("WARN", "vpn.connect_refused", name, s.f("cli.audit.connect_failed", name, err))
			return err
		}
	}
	ctx, done := s.trackConnect(ctx, name)
	defer done()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.onIntent != nil {
		s.onIntent(name, false)
	}

	aCfg := vpn.AdapterConfig{
		Host:          vpnCfg.Host,
		Port:          vpnCfg.Port,
		TunnelName:    vpnCfg.TunnelName,
		CertPath:      vpnCfg.Auth.Cert,
		KeyPath:       vpnCfg.Auth.Key,
		ConfigPath:    vpnCfg.ConfigFile,
		DefaultEgress: vpnCfg.DefaultEgress,
		AppRouteMode:  vpnCfg.AppRouteMode,
		Username:      vpnCfg.Auth.Username,
		Extra:         vpnCfg.AdapterExtra(),
		// Provider adapters (attach mode) call this when they start waiting
		// for the user to connect in the provider's app.
		OnAttach: func(st vpn.AttachState) { s.notifyAttach(name, st) },
	}

	if vpnCfg.Auth.UsernameKeychain != "" {
		u, err := config.GetCredential(name, "username")
		if err == nil {
			aCfg.Username = u
		}
	}
	if vpnCfg.Auth.PasswordKeychain != "" {
		p, err := config.GetCredential(name, "password")
		if err == nil {
			aCfg.Password = p
		}
	}

	if vpnCfg.AppRouteMode != "" {
		aCfg.Password = ""
		if !privops.WGAppRoutingSupported() {
			return errors.New(s.t("cli.app_route.unsupported"))
		}
		if adapter.Status().Normalize() == vpn.StatusConnected {
			type appRouteReporter interface{ AppRoutingActive() bool }
			if reporter, ok := adapter.(appRouteReporter); !ok || !reporter.AppRoutingActive() {
				return errors.New(s.f("cli.app_route.reconnect", name))
			}
		}
		paths := s.appRoutePaths(name)
		if len(paths) == 0 {
			return errors.New(s.f("cli.app_route.paths", name))
		}
		for _, path := range paths {
			if !filepath.IsAbs(path) {
				return errors.New(s.f("cli.app_route.paths", name))
			}
		}
		aCfg.AppExecutables = paths
	}

	if err := s.runProfilePreflight(name, vpnCfg, aCfg); err != nil {
		aCfg.Password = ""
		s.audit("ERROR", "vpn.preflight_failed", name, s.f("cli.audit.preflight_failed", name, err))
		return err
	}

	if vpnCfg.Type == "wireguard" && aCfg.ConfigPath != "" {
		// Let PolicyResolver own the OS routes for destinations. If they
		// are installed by the WireGuard client during startup, they
		// cannot be removed when a rule is paused.
		if cidrs := s.policyAllowedIPs(name, s.policyResolver != nil); len(cidrs) > 0 {
			if aCfg.TunnelName == "" {
				base := filepath.Base(aCfg.ConfigPath)
				aCfg.TunnelName = strings.TrimSuffix(base, filepath.Ext(base))
			}
			patched, err := patchWireGuardAllowedIPs(aCfg.ConfigPath, aCfg.TunnelName, cidrs)
			if err != nil {
				aCfg.Password = ""
				return errors.New(s.f("cli.policy.warn.patch_allowed_ips", name, err))
			} else {
				aCfg.ConfigPath = patched
				defer os.RemoveAll(filepath.Dir(patched))
			}
		}
	}

	if err := ctx.Err(); err != nil {
		aCfg.Password = ""
		return err
	}
	if adapter.Status().Normalize() != vpn.StatusConnected {
		if attachOnly {
			aCfg.Password = ""
			return ErrTunnelNotAttached
		}
		err := adapter.Connect(ctx, aCfg)
		aCfg.Password = ""
		if errors.Is(err, privops.ErrWGDefaultEgressConflict) {
			err = errors.New(s.t("cli.routes.default.conflict"))
		}
		if errors.Is(err, vpn.ErrUserActionTimeout) {
			err = fmt.Errorf("%s: %w", s.f("cli.attach.wait_timeout", name), err)
		}
		if err != nil {
			s.maybeEmitReauthAssistance(name, err)
			s.audit("ERROR", "vpn.connect_failed", name, s.f("cli.audit.connect_failed", name, err))
			return err
		}
	}
	// Cancelled by Disconnect (`down`) while connecting: never re-arm the
	// watchdog or install routes for a profile the user just turned off.
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.watchdog != nil {
		s.watchdog.MarkActive(name)
	}

	if s.dnsMgr != nil && s.cfg.Security.DNSGuard.Enabled && vpnCfg.Type != "wireguard" {
		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			dnsServers := info.DNS
			if len(dnsServers) == 0 && s.cfg.Security.DNSGuard.FallbackDNS != "" {
				if ip := net.ParseIP(s.cfg.Security.DNSGuard.FallbackDNS); ip != nil {
					dnsServers = []net.IP{ip}
				}
			}
			if len(dnsServers) > 0 {
				s.dnsMgr.OnConnect(name, info.InterfaceName, dnsServers)
			}
		}
	}

	// Policy routes for every adapter type: WireGuard also gets AllowedIPs;
	// the others get OS routes through the tunnel interface.
	if s.policyResolver != nil {
		var err error
		if vpnCfg.Type == "wireguard" {
			err = s.policyResolver.RegisterProfile(name, interfaceFromWGConfig(vpnCfg), vpnCfg.ConfigFile)
		} else if info, ierr := adapter.TunnelInfo(); ierr == nil && info != nil && info.InterfaceName != "" {
			err = s.policyResolver.RegisterTunnel(name, info.InterfaceName, info.DNS)
		}
		if err != nil {
			s.stderr(s.f("cli.policyresolver.warn", name, err))
		}
	}
	if s.applyKill != nil {
		_ = s.applyKill()
	}

	s.audit("INFO", "vpn.connect", name, s.f("cli.audit.connect", name))
	return nil
}

// CheckFullTunnelConflict returns a *FullTunnelConflictError when profile
// name carries all traffic and another tunnel already does (a connected
// Kongtrol profile or a provider app outside Kongtrol). Two full tunnels
// fight over the default route and, with WireGuard for Windows or a
// provider kill switch, block each other's traffic (ADR-0099). It also
// refuses, split tunnel or not, a direct Proton profile while Proton's app
// is connected and a direct profile whose tunnel address a connected one
// already uses (ADR-0103). Every refusal wraps ErrFullTunnelConflict, so
// the watchdog waits for the user instead of retrying.
func (s *ProfileService) CheckFullTunnelConflict(name string) error {
	if s == nil || s.cfg == nil {
		return nil
	}
	v, ok := s.cfg.VPNs[name]
	if !ok {
		return nil
	}
	var external []ExternalTunnel
	probed := false
	listExternal := func() []ExternalTunnel {
		if !probed && s.externalTunnels != nil {
			external, probed = s.externalTunnels(), true
		}
		return external
	}
	// A direct profile from a Proton account next to Proton's own app:
	// same account and tunnel address, split or not (ADR-0103).
	if isDirectProton(v) {
		if e, up := protonAppTunnel(listExternal(), connectedIfaces(s.adapters, name)); up {
			return newProtonAppConflictError(name, e)
		}
	}
	// Two direct profiles with one tunnel address (ADR-0103).
	if other, addr := sharedAddress(s.cfg, s.adapters, name, v); other != "" {
		return newSharedAddressError(name, other, addr)
	}
	if !ProfileCarriesAllTraffic(v) {
		return nil
	}
	managed, ifaces := managedFullTunnels(s.cfg, s.adapters, name)
	external = listExternal()
	if c := fullTunnelConflicts(name, v.Type, true, managed, ifaces, external); len(c) > 0 {
		return newFullTunnelConflictError(name, c[0])
	}
	return nil
}

func (s *ProfileService) Disconnect(ctx context.Context, name string) error {
	return s.disconnect(ctx, name, s.beginDisconnect)
}

// DisconnectForShutdown stops the adapter even if the helper is unavailable.
func (s *ProfileService) DisconnectForShutdown(ctx context.Context, name string) error {
	return s.disconnect(ctx, name, nil)
}

func (s *ProfileService) disconnect(ctx context.Context, name string, begin func(string) (func(), error)) error {
	adapter, ok := s.adapters.Get(name)
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.unknown_profile", name))
	}
	var finish func()
	if begin != nil {
		var err error
		finish, err = begin(name)
		if err != nil {
			if finish != nil {
				finish()
			}
			return fmt.Errorf("%s: %w", s.t("cli.policy.disconnect_protection_failed"), err)
		}
	}
	defer func() {
		if finish != nil {
			finish()
		}
	}()
	if s.watchdog != nil {
		s.watchdog.MarkIntended(name) // also cancels a watchdog reconnect/failover in flight
	}
	if s.onIntent != nil {
		s.onIntent(name, true) // turned off by the user: never a kill switch block
	}
	// Stop any connect still running for this profile (an `up` waiting for
	// the user in a provider app, a scheduler or policy connect) and wait
	// for it, so the status is final when `down` reports success and no
	// app window opens afterwards.
	if s.cancelConnects(name) && s.watchdog != nil {
		// A connect that finished just before the cancel may have
		// re-armed the watchdog: mark the intent again.
		s.watchdog.MarkIntended(name)
	}
	if err := adapter.Disconnect(ctx); err != nil {
		if errors.Is(err, vpn.ErrManualDisconnect) {
			// Provider app without a CLI: the user disconnects in the app
			// (the adapter opened it). Returned as-is so callers show the
			// waiting state; cleanup runs once the tunnel drops.
			s.manualDisconnect(name, adapter, finish)
			finish = nil
		}
		return err
	}
	s.afterDisconnect(name)
	return nil
}

// trackConnect registers a running Connect for name; Disconnect cancels
// the returned context. Call the returned func when Connect returns.
func (s *ProfileService) trackConnect(ctx context.Context, name string) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	p := &pendingConnect{cancel: cancel, done: make(chan struct{})}
	s.mu.Lock()
	if s.inflight == nil {
		s.inflight = make(map[string]map[uint64]*pendingConnect)
	}
	if s.inflight[name] == nil {
		s.inflight[name] = make(map[uint64]*pendingConnect)
	}
	s.nextID++
	id := s.nextID
	s.inflight[name][id] = p
	s.mu.Unlock()
	return ctx, func() {
		s.mu.Lock()
		delete(s.inflight[name], id)
		if len(s.inflight[name]) == 0 {
			delete(s.inflight, name)
		}
		s.mu.Unlock()
		cancel()
		close(p.done)
	}
}

// cancelConnects cancels every running Connect for name and waits until
// they returned (bounded by disconnectCancelWait). It reports whether any
// was running.
func (s *ProfileService) cancelConnects(name string) bool {
	s.mu.Lock()
	pending := make([]*pendingConnect, 0, len(s.inflight[name]))
	for _, p := range s.inflight[name] {
		pending = append(pending, p)
	}
	s.mu.Unlock()
	if len(pending) == 0 {
		return false
	}
	for _, p := range pending {
		p.cancel()
	}
	deadline := time.NewTimer(disconnectCancelWait)
	defer deadline.Stop()
	for _, p := range pending {
		select {
		case <-p.done:
		case <-deadline.C:
			return true
		}
	}
	return true
}

// notifyAttach reports a provider adapter's attach phase change: the alert
// feed and audit log get a one-line message (daemon language), and
// onAttach (e.g. a collector refresh) lets live views show it at once.
// Live views render the phase themselves from TunnelMetrics.Attach in
// their own language.
func (s *ProfileService) notifyAttach(name string, st vpn.AttachState) {
	key, args := st.MessageKey()
	if key == "" {
		return
	}
	msg := name + ": " + s.f(key, args...)
	level := "INFO"
	if st.Phase == vpn.PhaseWaitingUser && st.Reason != "" {
		level = "WARN"
	}
	if s.emitAlert != nil {
		s.emitAlert(level, name, msg)
	}
	s.audit(level, "vpn.attach."+string(st.Phase), name, msg)
	if s.onAttach != nil {
		s.onAttach(name, st)
	}
}

// manualDisconnect handles a provider tunnel the user must close in the
// app (adapter returned vpn.ErrManualDisconnect): tell the user, and run
// the post-disconnect cleanup (DNS, kill switch, audit) in the background
// once the tunnel drops, bounded by the profile's wait_timeout. It does
// not block: callers show the waiting state and poll the tunnel status.
func (s *ProfileService) manualDisconnect(name string, adapter vpn.VPNAdapter, finish func()) {
	st := vpn.AttachState{Phase: vpn.PhaseWaitingDisconnect, App: name, Since: time.Now()}
	if r, ok := adapter.(vpn.AttachReporter); ok {
		if cur := r.AttachState(); cur != nil {
			st = *cur
		}
	}
	s.notifyAttach(name, st)

	timeout := 5 * time.Minute
	if d, err := time.ParseDuration(s.cfg.VPNs[name].WaitTimeout); err == nil && d > 0 {
		timeout = d
	}
	go func() {
		if finish != nil {
			defer finish()
		}
		deadline := time.Now().Add(timeout)
		for adapter.Status().Normalize() == vpn.StatusConnected {
			if time.Now().After(deadline) {
				s.audit("WARN", "vpn.disconnect_timeout", name, s.f("cli.attach.disconnect_timeout", name))
				return
			}
			time.Sleep(time.Second)
		}
		s.afterDisconnect(name)
	}()
}

// afterDisconnect is the cleanup shared by Disconnect and manualDisconnect.
func (s *ProfileService) afterDisconnect(name string) {
	if s.policyResolver != nil {
		s.policyResolver.UnregisterProfile(name)
	}
	if s.dnsMgr != nil {
		s.dnsMgr.OnDisconnect(name)
	}
	if s.applyKill != nil {
		_ = s.applyKill()
	}
	s.audit("INFO", "vpn.disconnect", name, s.f("cli.audit.disconnect", name))
}

func (s *ProfileService) HandleReconnectError(profile string, err error) {
	s.maybeEmitReauthAssistance(profile, err)
}

func (s *ProfileService) PolicyAllowedIPs(profileName string) []string {
	return s.policyAllowedIPs(profileName, false)
}

func (s *ProfileService) appRoutePaths(name string) []string {
	engine, err := policy.New(s.cfg)
	if err != nil {
		return nil
	}
	var paths []string
	for _, rule := range engine.Rules() {
		if rule.Via != name || len(rule.Match.Domains) != 0 || len(rule.Match.IPRanges) != 0 {
			continue
		}
		for _, executable := range rule.Match.Apps {
			if !slices.Contains(paths, executable) {
				paths = append(paths, executable)
			}
		}
	}
	return paths
}

func (s *ProfileService) runProfilePreflight(name string, vpnCfg config.VPNConfig, aCfg vpn.AdapterConfig) error {
	if vpnCfg.DefaultEgress || vpnCfg.AppRouteMode != "" {
		if status := vpnCfg.DefaultEgressStatus(); !status.Compatible {
			key := "config.err.default_egress_" + status.Reason
			return fmt.Errorf("%s %s", s.t(key), s.t(key+".fix"))
		}
	}
	if vpnCfg.ConfigFile != "" {
		if err := config.CheckFileReadable(vpnCfg.ConfigFile); err != nil {
			// Only root can read it (e.g. /etc/wireguard): say so and how
			// to copy it, never a bare "permission denied" (ADR-0058).
			return fmt.Errorf("%s", s.f("cli.preflight.config_root_only", name, vpnCfg.ConfigFile))
		}
		if _, err := os.Stat(vpnCfg.ConfigFile); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_config", name, err))
		}
	}
	if vpnCfg.Auth.Cert != "" {
		if _, err := os.Stat(vpnCfg.Auth.Cert); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_cert", name, err))
		}
	}
	if vpnCfg.Auth.Key != "" {
		if _, err := os.Stat(vpnCfg.Auth.Key); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_key", name, err))
		}
	}
	if vpnCfg.Auth.PasswordKeychain != "" && strings.TrimSpace(aCfg.Password) == "" {
		return fmt.Errorf("%s", s.f("cli.preflight.missing_password", name))
	}
	// Consumer provider adapters (protonvpn, nordvpn, surfshark) need no
	// CLI: without one they open the app and wait for the user (attach
	// mode), so a missing CLI is neither an error nor worth a warning.
	consumerApp := config.ProviderTypes[vpnCfg.Type] && vpnCfg.Type != "forticlient"
	if vpnCfg.Type == "openvpn" {
		if err := s.openvpnPreflight(name, vpnCfg); err != nil {
			return err
		}
	} else if vpnCfg.BinaryPath != "" {
		if _, err := os.Stat(vpnCfg.BinaryPath); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_binary", name, vpnCfg.BinaryPath))
		}
	} else if vpndetect.Known(vpnCfg.Type) && !consumerApp {
		// Check the exact locations the adapter executes on this OS (e.g.
		// wireguard.exe under Program Files on Windows, wg-quick on PATH on
		// Linux) instead of Linux binary names on PATH everywhere.
		if _, ok := vpndetect.FindStandard(vpnCfg.Type); !ok {
			s.warn(s.f("cli.preflight.binary_not_found_warning", name,
				strings.Join(vpndetect.StandardLocations(vpnCfg.Type), ", ")))
		}
	}
	if aCfg.Host != "" {
		port := aCfg.Port
		if port == 0 {
			port = 443
		}
		dialer := net.Dialer{Timeout: 2 * time.Second}
		conn, err := dialer.Dial("tcp", net.JoinHostPort(aCfg.Host, strconv.Itoa(port)))
		if err != nil {
			s.warn(s.f("cli.preflight.port_warning", name, aCfg.Host, port, err))
		} else {
			_ = conn.Close()
		}
	}
	return nil
}

// openvpnPreflight replaces the generic binary check for OpenVPN: the binary
// is usually not on PATH (Windows installer, Homebrew sbin), openvpn creates
// the tunnel device and routes itself so it needs admin/root, and a second
// instance of a profile already running (e.g. in OpenVPN GUI) would fight
// over the same routes. Each case gets an actionable, translated message
// instead of openvpn's raw console error.
func (s *ProfileService) openvpnPreflight(name string, vpnCfg config.VPNConfig) error {
	if !privops.IsElevated() {
		// The helper runs its own trusted openvpn; it only has to answer.
		// The error keeps the privops sentinel so clients render the
		// translated "install the helper once" line (GuidanceKey).
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := privops.Available(ctx)
		cancel()
		if err != nil {
			key, _ := privops.GuidanceKey(err)
			if key == "" {
				key = "service.error.not_installed"
			}
			return fmt.Errorf("%s (%w)", s.t(key), err)
		}
	} else if _, err := openvpn.FindBinary(vpnCfg.BinaryPath); err != nil {
		if vpnCfg.BinaryPath != "" {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_binary", name, vpnCfg.BinaryPath))
		}
		return fmt.Errorf("%s", s.f("cli.preflight.openvpn_not_found", name))
	}
	if openvpn.RunningWithConfig(vpnCfg.ConfigFile) {
		return fmt.Errorf("%s", s.f("cli.preflight.openvpn_already_running", name, filepath.Base(vpnCfg.ConfigFile)))
	}
	return nil
}

func (s *ProfileService) maybeEmitReauthAssistance(profile string, err error) {
	if err == nil || !isReauthError(err) {
		return
	}
	vpnCfg, ok := s.cfg.VPNs[profile]
	if !ok {
		return
	}
	hint := s.reauthHint(vpnCfg)
	msg := s.f("cli.alert.reauth_required", profile, err)
	if s.emitAlert != nil {
		s.emitAlert("WARN", profile, msg)
		if hint != "" {
			s.emitAlert("INFO", profile, s.f("cli.alert.reauth_hint", profile, hint))
		}
	}
	s.audit("WARN", "vpn.reauth_required", profile, msg)
}

func isReauthError(err error) bool {
	msg := strings.ToLower(err.Error())
	needles := []string{
		"unauthorized", "authorization", "authentication", "auth failed",
		"invalid credentials", "credential", "password", "token expired",
		"session expired", "login failed", "mfa",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func (s *ProfileService) reauthHint(vpnCfg config.VPNConfig) string {
	switch vpnCfg.Type {
	case "forticlient", "globalprotect", "ciscoanyconnect":
		return s.t("cli.reauth.hint.gui")
	case "openvpn", "wireguard":
		return s.t("cli.reauth.hint.credentials")
	case "protonvpn", "nordvpn", "cloudflarewarp", "tailscale":
		return s.t("cli.reauth.hint.session")
	default:
		return s.t("cli.reauth.hint.generic")
	}
}

func (s *ProfileService) policyAllowedIPs(profileName string, bootstrap bool) []string {
	vpnCfg, ok := s.cfg.VPNs[profileName]
	if !ok || vpnCfg.DefaultEgress || vpnCfg.AppRouteMode != "" {
		return nil
	}

	seen := make(map[string]struct{})
	var cidrs []string
	add := func(cidr string) {
		if cidr == "" {
			return
		}
		if _, ok := seen[cidr]; !ok {
			seen[cidr] = struct{}{}
			cidrs = append(cidrs, cidr)
		}
	}

	var endpointIP net.IP
	if vpnCfg.ConfigFile != "" {
		endpointIP, _ = wireguard.ParseEndpoint(vpnCfg.ConfigFile)
		if addr, err := wireguard.ParseConfigAddress(vpnCfg.ConfigFile); err == nil && addr != nil {
			if v4 := addr.To4(); v4 != nil {
				add(fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2]))
			} else {
				add(addr.String() + "/128")
			}
		}
		for _, dns := range wireguard.ParseConfigDNS(vpnCfg.ConfigFile) {
			if dns.To4() != nil {
				add(dns.String() + "/32")
			} else if dns.To16() != nil {
				add(dns.String() + "/128")
			}
		}
	}
	if bootstrap && len(cidrs) > 0 {
		return cidrs
	}

	hasDomains := false
	for _, pol := range s.cfg.Policies {
		if pol.Disabled || pol.Via != profileName {
			continue
		}
		for _, cidr := range pol.Match.IPRanges {
			add(cidr)
		}
		for _, domain := range pol.Match.Domains {
			hasDomains = true
			var lookups []string
			if strings.HasPrefix(domain, "*.") {
				base := strings.TrimPrefix(domain, "*.")
				lookups = append(lookups, base)
				for _, prefix := range []string{"www.", "api.", "cdn.", "app.", "docs.", "console."} {
					lookups = append(lookups, prefix+base)
				}
			} else {
				lookups = []string{domain}
			}

			for _, lookup := range lookups {
				ips, err := net.LookupHost(lookup)
				if err != nil {
					continue
				}
				for _, ip := range ips {
					parsed := net.ParseIP(ip)
					if parsed == nil {
						continue
					}
					if endpointIP != nil && parsed.Equal(endpointIP) {
						continue
					}
					if parsed.To4() != nil {
						add(ip + "/32")
					}
				}
			}
		}
	}

	if !hasDomains && len(cidrs) == 0 {
		return nil
	}
	return cidrs
}

func interfaceFromWGConfig(vpnCfg config.VPNConfig) string {
	if vpnCfg.TunnelName != "" {
		return vpnCfg.TunnelName
	}
	base := filepath.Base(vpnCfg.ConfigFile)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func patchWireGuardAllowedIPs(configPath, tunnelName string, cidrs []string) (string, error) {
	original, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}

	allowedLine := "AllowedIPs = " + strings.Join(cidrs, ", ")
	var out strings.Builder
	inPeer := false
	replaced := false
	for _, line := range strings.Split(string(original), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPeer = strings.EqualFold(trimmed, "[Peer]")
			replaced = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(trimmed), "allowedips") {
			if !replaced {
				out.WriteString(allowedLine + "\n")
				replaced = true
			}
			continue
		}
		out.WriteString(line + "\n")
	}

	tmpDir, err := os.MkdirTemp("", "kongtrol-wg-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	tmpPath := filepath.Join(tmpDir, tunnelName+".conf")
	if err := os.WriteFile(tmpPath, []byte(out.String()), 0o600); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("write temp config: %w", err)
	}
	return tmpPath, nil
}

func (s *ProfileService) f(key string, args ...any) string {
	if s.format == nil {
		return key
	}
	return s.format(key, args...)
}

func (s *ProfileService) t(key string) string {
	if s.translate == nil {
		return key
	}
	return s.translate(key)
}

func (s *ProfileService) audit(level, action, profile, message string) {
	if s.logAudit != nil {
		s.logAudit(level, action, profile, message)
	}
}

func (s *ProfileService) warn(message string) {
	if s.emitWarn != nil {
		s.emitWarn(message)
	}
}

func (s *ProfileService) stderr(message string) {
	if s.emitStderr != nil {
		s.emitStderr(message)
	}
}
