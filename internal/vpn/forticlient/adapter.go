// Package forticlient implements the VPNAdapter interface for FortiClient
// in attach mode (internal/vpn/appattach): the tunnel is found by its
// network interface however it came up.
//
// Automatic connect (step 2, best effort) depends on the installed version
// (DetectInstall + SelectPlan, docs/forticlient-support.md): FortiVPN.exe
// CLI + native auth dialog, GUI automation of the 6.4 window, the Linux /
// macOS CLI, or none. When it fails (e.g. a FortiClient version changed the
// window layout) Kongtrol opens FortiClient and waits for the user to
// connect there, which works on any version. Disconnect uses the same
// strategy; if the tunnel is still up the user is asked to disconnect in
// FortiClient.
package forticlient

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/netstats"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func init() {
	vpn.Register("forticlient", func() vpn.VPNAdapter { return &Adapter{} })
	appattach.RegisterMatcher("forticlient", matcher)
}

const (
	// autoWait bounds the wait for the tunnel after automation (as before
	// attach mode: FortiClient may show MFA / certificate prompts).
	autoWait = 90 * time.Second
	// disconnectGrace is how long GUI / CLI disconnect gets before the
	// user is asked to disconnect in FortiClient.
	disconnectGrace = 8 * time.Second
)

// installTTL is how long a DetectInstall result is reused (Capabilities
// and Status build the provider often; FortiClient upgrades are rare).
const installTTL = time.Minute

// Adapter implements vpn.VPNAdapter for FortiClient.
type Adapter struct {
	mu      sync.RWMutex
	lastCfg vpn.AdapterConfig
	sess    appattach.Session

	instMu  sync.Mutex
	inst    Install
	instFor string // app_path the cached install was detected for
	instAt  time.Time
}

func (a *Adapter) Name() string { return "forticlient" }

// Version is the installed FortiClient's ProductVersion on Windows, or
// "unknown".
func (a *Adapter) Version() string {
	if v := a.install("").Version; v != "" {
		return v
	}
	return "unknown"
}

// install returns DetectInstall(appPath), cached for installTTL.
func (a *Adapter) install(appPath string) Install {
	a.instMu.Lock()
	defer a.instMu.Unlock()
	if a.instAt.IsZero() || a.instFor != appPath || time.Since(a.instAt) > installTTL {
		a.inst, a.instFor, a.instAt = DetectInstall(appPath), appPath, time.Now()
	}
	return a.inst
}

// Configure pre-seeds the profile config so Status() attaches to a tunnel
// connected in FortiClient.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	cfg.Password = ""
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastCfg = cfg
}

// Capabilities: FortiClient manages its own routing and is exclusive. The
// version's automation strategy is the automatic reconnect path, so
// auto_connect: false, or a version with no strategy (StrategyAttach),
// makes the watchdog notify instead.
func (a *Adapter) Capabilities() vpn.Capabilities {
	p, o := a.provider("")
	return vpn.Capabilities{
		SupportsSplitTunnel: false,
		SupportsMultiConn:   false,
		SupportsReconnect:   true,
		ManualReconnect:     p.ManualReconnect(o),
	}
}

// provider builds the attach-mode description for the installed version's
// plan (SelectPlan); password is only set during Connect and captured by
// the AutoConnect closure for that call.
func (a *Adapter) provider(password string) (appattach.Provider, appattach.Options) {
	a.mu.RLock()
	cfg := a.lastCfg
	a.mu.RUnlock()
	o := appattach.ParseOptions(cfg.Extra)
	allowInsecureCert, _ := strconv.ParseBool(cfg.Extra["allow_insecure_cert"])
	inst := a.install(o.AppPath)
	p := appattach.Provider{
		App:            app,
		Matcher:        matcher.WithOverride(o.Interface),
		Location:       cfg.TunnelName,
		AutoWait:       autoWait,
		DisconnectWait: 2 * time.Second,
	}
	m := p.Matcher
	tunnelUp := func() bool { _, ok := a.sess.Detect(m); return ok }

	switch SelectPlan(runtime.GOOS, inst).Strategy {
	case StrategyGUI64:
		p.AutoConnect = func(ctx context.Context) error {
			return guiConnect(ctx, inst.Path, cfg.TunnelName, cfg.Username, password, allowInsecureCert)
		}
		p.Disconnect = func(ctx context.Context) error {
			return a.disconnectWith(ctx, m, func(ctx context.Context) {
				_ = guiDisconnect(ctx, inst.Path)
				_ = disconnectV6(inst.Path, cfg.TunnelName)
			})
		}
	case StrategyFortiVPN:
		p.AutoConnect = func(ctx context.Context) error {
			return fortiVPNConnect(ctx, inst, cfg.TunnelName, cfg.Username, password, allowInsecureCert, tunnelUp)
		}
		p.Disconnect = func(ctx context.Context) error {
			return a.disconnectWith(ctx, m, func(ctx context.Context) {
				_ = fortiVPNDisconnect(ctx, inst.FortiVPN, cfg.TunnelName)
			})
		}
	case StrategyCLI:
		p.AutoConnect = func(context.Context) error {
			return connectV6(inst.Path, cfg.Host, cfg.Port, cfg.CertPath, cfg.KeyPath, cfg.Username, password)
		}
		p.Disconnect = func(ctx context.Context) error {
			return a.disconnectWith(ctx, m, func(context.Context) { _ = disconnectV6(inst.Path, cfg.TunnelName) })
		}
	default: // StrategyAttach: open FortiClient and wait for the user.
	}
	return p, o
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	password := cfg.Password
	cfg.Password = "" // never kept in lastCfg
	a.mu.Lock()
	a.lastCfg = cfg
	a.mu.Unlock()
	p, o := a.provider(password)
	return a.sess.Connect(ctx, p, o, cfg.OnAttach)
}

// disconnectWith runs the strategy's disconnect steps, then waits for the
// tunnel to drop. An error means the tunnel is still up; attach mode then
// shows the translated "disconnect in FortiClient" guidance and waits.
// Force-disabling the network adapter was dropped: it needs administrator
// rights, and the privileged helper only runs a closed list of operations
// (no "disable any adapter" op).
func (a *Adapter) disconnectWith(ctx context.Context, m appattach.Matcher, steps func(context.Context)) error {
	steps(ctx)
	if a.sess.WaitDown(ctx, m, disconnectGrace) == nil {
		return nil
	}
	if _, ok := a.sess.Detect(m); !ok {
		return nil
	}
	return errors.New("forticlient: automated disconnect did not drop the tunnel")
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	p, o := a.provider("")
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

// Status reports the tunnel state by interface detection, so a tunnel
// connected in the FortiClient GUI (or by another kongtrol process) is
// attached too.
func (a *Adapter) Status() vpn.Status {
	p, o := a.provider("")
	return a.sess.Status(p, p.ManualReconnect(o)).Normalize()
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	if a.Status() != vpn.StatusConnected {
		return nil, nil
	}
	info := a.sess.Info()
	if info == nil || info.InterfaceName == "" {
		return info, nil
	}
	// Byte counters and the DNS servers FortiClient pushed, from the OS.
	info.BytesSent, info.BytesReceived, _ = netstats.Counters(info.InterfaceName)
	info.DNS = ifaceDNSServers(info.InterfaceName)
	return info, nil
}

// AttachState implements vpn.AttachReporter.
func (a *Adapter) AttachState() *vpn.AttachState { return a.sess.AttachState() }

// app is FortiClient per OS (`app_path:` overrides).
var app = appattach.App{
	DisplayName:  "FortiClient",
	WindowsPaths: []string{`C:\Program Files\Fortinet\FortiClient\FortiClient.exe`},
	MacApp:       "FortiClient",
	LinuxCmds:    []string{"forticlient"},
}

// matcher finds the FortiClient SSL VPN tunnel (same rules as before attach
// mode): Unix names fortissl*, vpnssl0, ssl0, ppp0; Windows adapters
// described "Fortinet SSL VPN Virtual Ethernet Adapter" / "Fortinet
// Virtual Ethernet Adapter (NDIS 6.30)". It must carry an IPv4 address
// (FortiClient keeps the adapter around while disconnected).
var matcher = appattach.Matcher{
	Patterns:    []string{"fortissl*", "vpnssl0*", "ssl0*", "ppp0*", "*fortinet*", "*sslvpn*", "*ssl vpn*"},
	RequireIPv4: true,
}
