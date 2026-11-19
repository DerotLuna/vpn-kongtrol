// Package nordvpn implements the VPNAdapter interface for the official
// NordVPN app in attach mode (internal/vpn/appattach): the tunnel is found
// by its network interface however it came up.
//
// Automatic connect / disconnect (step 2 of attach mode), per OS:
//   - Windows: the CLI switches of the app executable
//     ("C:\Program Files\NordVPN\NordVPN.exe" -c / -c -g <group> /
//     -c -n <server> / -d).
//   - Linux: the official `nordvpn` CLI (connect / disconnect / status).
//     The user must have run `nordvpn login` once.
//   - macOS: no official CLI, so the app is opened and Kongtrol waits for
//     the user; the tunnel (a generic utunN) is found through the app's
//     network service in `scutil --nc list`.
//
// Target selection: `server` (exact server: "United States #3710" on
// Windows, "us3710" on Linux), `group` ("P2P", "Dedicated IP", ...),
// `country` / `city`; all empty = quick connect. Older profiles that put a
// country / group into `server` keep working. `protocol` is ignored: the
// NordVPN client keeps its own technology setting.
package nordvpn

import (
	"context"
	"errors"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// ErrUnsupportedOS reports that the NordVPN client ships no CLI on this OS
// (macOS). It is not a Connect error: attach mode opens the app and waits
// for the user instead. Kept for callers (doctor, wizard) that explain it.
var ErrUnsupportedOS = errors.New("nordvpn: the NordVPN client has no CLI on this OS; Kongtrol opens the app and waits for you to connect")

// ErrNotLoggedIn is returned when the Linux CLI reports no active session.
var ErrNotLoggedIn = errors.New("nordvpn: not logged in — run `nordvpn login` once, then retry")

func init() {
	vpn.Register("nordvpn", func() vpn.VPNAdapter { return &Adapter{} })
	appattach.RegisterMatcher("nordvpn", matcher)
}

// Adapter implements vpn.VPNAdapter for the official NordVPN client.
type Adapter struct {
	mu      sync.RWMutex
	lastCfg vpn.AdapterConfig
	sess    appattach.Session
}

func (a *Adapter) Name() string { return "nordvpn" }

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

// provider builds the attach-mode description for the current config.
func (a *Adapter) provider() (appattach.Provider, appattach.Options) {
	a.mu.RLock()
	extra := a.lastCfg.Extra
	a.mu.RUnlock()
	o := appattach.ParseOptions(extra)
	bin := extra["binary_path"]
	t := targetFrom(extra)
	p := appattach.Provider{App: appFor(bin), Matcher: matcher.WithOverride(o.Interface), Location: t.label()}
	if CheckPlatform() == nil {
		p.AutoConnect = func(context.Context) error { return connect(bin, t) }
		p.Disconnect = func(context.Context) error { return disconnect(bin) }
	}
	return p, o
}

// Capabilities are deliberately conservative: the NordVPN client's own
// split tunneling and kill switch are configured in the app, not driven by
// Kongtrol, and it runs a single tunnel at a time. Without a CLI (macOS)
// or with auto_connect: false only the user can reconnect.
func (a *Adapter) Capabilities() vpn.Capabilities {
	p, o := a.provider()
	return vpn.Capabilities{
		SupportsReconnect: true,
		ManualReconnect:   p.ManualReconnect(o),
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	cfg.Password = "" // the session lives in the NordVPN client
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

// Status reports the tunnel state by interface detection, which works on
// every OS whether or not a NordVPN CLI exists (so a tunnel started from
// the NordVPN app is attached too).
func (a *Adapter) Status() vpn.Status {
	p, o := a.provider()
	return a.sess.Status(p, p.ManualReconnect(o)).Normalize()
}

// TunnelInfo merges interface detection (name, assigned IP) with whatever
// the CLI adds (server IP on Linux).
func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	if a.Status() != vpn.StatusConnected {
		return nil, nil
	}
	info := a.sess.Info()
	if info == nil {
		return nil, nil
	}
	if CheckPlatform() == nil {
		a.mu.RLock()
		override := a.lastCfg.Extra["binary_path"]
		a.mu.RUnlock()
		if extra, err := cliTunnelInfo(override); err == nil && extra != nil {
			info.RemoteIP = extra.RemoteIP
		}
	}
	return info, nil
}

// AttachState implements vpn.AttachReporter.
func (a *Adapter) AttachState() *vpn.AttachState { return a.sess.AttachState() }
