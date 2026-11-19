// Package protonvpn implements the VPNAdapter interface for the official
// Proton VPN app in attach mode (internal/vpn/appattach): the tunnel is
// found by its network interface however it came up.
//
// Automatic connect / disconnect (step 2 of attach mode) exists only on
// Linux, through Proton's official CLI: the current `protonvpn` CLI
// (protonvpn.com/support/use-linux-cli) when installed, else the legacy
// `protonvpn-cli` v3. Sign in once first (`protonvpn signin <user>` or
// `protonvpn-cli login`). Windows and macOS have no Proton CLI: `up` opens
// the Proton VPN app and waits for the user to connect there.
//
// Target selection: `server` (exact server, e.g. "CH#242"), `city`,
// `country` (code such as "CH" or a name); all empty = fastest. Older
// profiles with `server: CH` / `server: fastest` keep working.
package protonvpn

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// ErrUnsupportedOS reports that Proton ships no CLI on this OS (Windows,
// macOS). It is not a Connect error: attach mode opens the app and waits
// for the user. Kept for callers (doctor, wizard) that explain it.
var ErrUnsupportedOS = errors.New("protonvpn: no Proton VPN CLI on this OS; Kongtrol opens the app and waits for you to connect")

func init() {
	vpn.Register("protonvpn", func() vpn.VPNAdapter { return &Adapter{} })
	appattach.RegisterMatcher("protonvpn", matcher)
}

// Adapter implements vpn.VPNAdapter for Proton VPN.
type Adapter struct {
	mu      sync.RWMutex
	lastCfg vpn.AdapterConfig
	sess    appattach.Session
}

func (a *Adapter) Name() string { return "protonvpn" }

// Configure pre-seeds the profile config so Version() honors binary_path
// and Status() attaches to a tunnel started in the app.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastCfg = cfg
}

func (a *Adapter) Version() string {
	a.mu.RLock()
	override := a.lastCfg.Extra["binary_path"]
	a.mu.RUnlock()
	if v := version(override); v != "" {
		return v
	}
	return "unknown"
}

// hasCLI reports whether a Proton CLI can drive the tunnel here.
func hasCLI(override string) bool {
	if CheckPlatform() != nil {
		return false
	}
	_, _, err := findCLI(override)
	return err == nil
}

// provider builds the attach-mode description for the current config.
func (a *Adapter) provider() (appattach.Provider, appattach.Options) {
	a.mu.RLock()
	extra := a.lastCfg.Extra
	a.mu.RUnlock()
	o := appattach.ParseOptions(extra)
	bin := extra["binary_path"]
	t := targetFrom(extra)
	p := appattach.Provider{App: app, Matcher: matcher.WithOverride(o.Interface), Location: t.label()}
	if hasCLI(bin) {
		protocol := extra["protocol"]
		p.AutoConnect = func(context.Context) error { return connect(bin, t, protocol) }
		p.Disconnect = func(context.Context) error { return disconnect(bin) }
	}
	return p, o
}

// Capabilities: Proton's own split tunneling is configured in the app.
// Without a CLI (Windows, macOS) or with auto_connect: false only the user
// can reconnect.
func (a *Adapter) Capabilities() vpn.Capabilities {
	p, o := a.provider()
	return vpn.Capabilities{
		SupportsSplitTunnel: true,
		SupportsReconnect:   true,
		ManualReconnect:     p.ManualReconnect(o),
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	cfg.Password = "" // the session lives in the Proton app / CLI
	a.mu.Lock()
	a.lastCfg = cfg
	a.mu.Unlock()
	p, o := a.provider()
	return a.sess.Connect(ctx, p, o, cfg.OnAttach)
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	p, o := a.provider()
	return a.sess.Disconnect(ctx, p, o)
}

func (a *Adapter) Reconnect(ctx context.Context) error {
	if err := a.Disconnect(ctx); err != nil {
		return err
	}
	a.mu.RLock()
	cfg := a.lastCfg
	a.mu.RUnlock()
	return a.Connect(ctx, cfg)
}

// Status reports the tunnel state by interface detection (a tunnel started
// from the Proton VPN app is attached too).
func (a *Adapter) Status() vpn.Status {
	p, o := a.provider()
	return a.sess.Status(p, p.ManualReconnect(o)).Normalize()
}

// TunnelInfo merges interface detection with what the Linux CLI adds
// (server IP).
func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	if a.Status() != vpn.StatusConnected {
		return nil, nil
	}
	info := a.sess.Info()
	if info == nil {
		return nil, nil
	}
	a.mu.RLock()
	override := a.lastCfg.Extra["binary_path"]
	a.mu.RUnlock()
	if hasCLI(override) {
		if ok, serverIP, _, err := connectionState(override); err == nil && ok && serverIP != "" {
			info.RemoteIP = net.ParseIP(serverIP)
		}
	}
	return info, nil
}

// AttachState implements vpn.AttachReporter.
func (a *Adapter) AttachState() *vpn.AttachState { return a.sess.AttachState() }

// label is the target for user messages ("CH#242", "Zurich", "CH"),
// empty for fastest.
func (t target) label() string {
	switch {
	case t.Server != "":
		return t.Server
	case t.City != "":
		return t.City
	case isCode(t.Country):
		return strings.ToUpper(t.Country)
	}
	return t.Country
}

// app is the official Proton VPN app per OS (locations best effort;
// `app_path:` overrides them). On Windows the same lookup as the detect
// list: ProtonVPN.Launcher.exe, else the client in the newest v<version>
// folder, in the default or registry install folder, else the legacy app.
var app = appattach.App{
	DisplayName: "Proton VPN",
	WindowsFind: vpndetect.ProtonVPNAppPaths,
	MacApp:      "Proton VPN",
	LinuxCmds:   []string{"protonvpn-app"},
}

// matcher finds the Proton tunnel: Linux proton0 / pvpn* (excluding the
// kill-switch / leak-guard dummies pvpnksintrf0, pvpnroutintrf0,
// ipv6leakintrf0 that stay up while disconnected); Windows adapters with
// "Proton" in the name or description ("TAP-ProtonVPN Windows Adapter V9";
// WireGuard adapter name best effort); macOS by the app's network service
// in `scutil --nc list` (best effort). `interface:` overrides all of this.
var matcher = appattach.Matcher{
	Patterns: []string{"proton*", "pvpn*", "*protonvpn*"},
	Exclude:  []string{"*intrf*"},
	Services: []string{"proton*"},
}
