// Package surfshark implements the VPNAdapter interface for Surfshark in
// attach mode (internal/vpn/appattach). Surfshark ships no command-line
// interface on Windows, macOS or (current) Linux, so Kongtrol never
// connects it by itself: `up` attaches to a tunnel that is already up, or
// opens the Surfshark app and waits for the user to connect there; `down`
// asks the user to disconnect in the app (vpn.ErrManualDisconnect), and
// the watchdog notifies instead of reconnecting. Location / server are
// chosen in the app.
package surfshark

import (
	"context"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func init() {
	vpn.Register("surfshark", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for the Surfshark app.
type Adapter struct {
	mu      sync.RWMutex
	lastCfg vpn.AdapterConfig
	sess    appattach.Session
}

func (a *Adapter) Name() string    { return "surfshark" }
func (a *Adapter) Version() string { return "app" }

// Configure pre-seeds the profile config (interface override, app_path)
// so Status() attaches to a tunnel started in the app.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastCfg = cfg
}

// Capabilities: Kongtrol cannot reconnect Surfshark (no CLI).
func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{ManualReconnect: true}
}

func (a *Adapter) provider() (appattach.Provider, appattach.Options) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	o := appattach.ParseOptions(a.lastCfg.Extra)
	return appattach.Provider{App: app, Matcher: matcher.WithOverride(o.Interface)}, o
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	cfg.Password = "" // the session lives in the Surfshark app
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

func (a *Adapter) Status() vpn.Status {
	p, _ := a.provider()
	return a.sess.Status(p, true).Normalize()
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	if a.Status() != vpn.StatusConnected {
		return nil, nil
	}
	return a.sess.Info(), nil
}

// AttachState implements vpn.AttachReporter.
func (a *Adapter) AttachState() *vpn.AttachState { return a.sess.AttachState() }
