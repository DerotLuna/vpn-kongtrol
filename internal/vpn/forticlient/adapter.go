// Package forticlient implements the VPNAdapter interface for FortiClient
// in attach mode (internal/vpn/appattach): the tunnel is found by its
// network interface however it came up.
//
// Automatic connect (step 2, best effort): on Windows GUI automation of the
// FortiClient window (EMS policies often block /vpnconnect); on Linux /
// macOS the forticlientsslvpn CLI. When it fails (e.g. a FortiClient
// version changed the window layout) Kongtrol opens FortiClient and waits
// for the user to connect there. Disconnect: GUI automation + CLI; if the
// tunnel is still up the user is asked to disconnect in FortiClient.
package forticlient

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/netstats"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func init() {
	vpn.Register("forticlient", func() vpn.VPNAdapter { return &Adapter{} })
}

const (
	// autoWait bounds the wait for the tunnel after automation (as before
	// attach mode: FortiClient may show MFA / certificate prompts).
	autoWait = 90 * time.Second
	// disconnectGrace is how long GUI / CLI disconnect gets before the
	// user is asked to disconnect in FortiClient.
	disconnectGrace = 8 * time.Second
)

// Adapter implements vpn.VPNAdapter for FortiClient.
type Adapter struct {
	mu      sync.RWMutex
	lastCfg vpn.AdapterConfig
	sess    appattach.Session
}

func (a *Adapter) Name() string { return "forticlient" }

func (a *Adapter) Version() string {
	out, _ := versionCmd()
	if out != "" {
		lines := strings.SplitN(out, "\n", 2)
		return strings.TrimSpace(lines[0])
	}
	return "6.4.x"
}

// Configure pre-seeds the profile config so Status() attaches to a tunnel
// connected in FortiClient.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	cfg.Password = ""
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastCfg = cfg
}

// Capabilities: FortiClient manages its own routing and is exclusive. GUI
// automation / CLI is the automatic reconnect path, so only
// auto_connect: false makes the watchdog notify instead.
func (a *Adapter) Capabilities() vpn.Capabilities {
	p, o := a.provider("")
	return vpn.Capabilities{
		SupportsSplitTunnel: false,
		SupportsMultiConn:   false,
		SupportsReconnect:   true,
		ManualReconnect:     p.ManualReconnect(o),
	}
}

// provider builds the attach-mode description; password is only set
// during Connect and captured by the AutoConnect closure for that call.
func (a *Adapter) provider(password string) (appattach.Provider, appattach.Options) {
	a.mu.RLock()
	cfg := a.lastCfg
	a.mu.RUnlock()
	o := appattach.ParseOptions(cfg.Extra)
	allowInsecureCert, _ := strconv.ParseBool(cfg.Extra["allow_insecure_cert"])
	p := appattach.Provider{
		App:      app,
		Matcher:  matcher.WithOverride(o.Interface),
		Location: cfg.TunnelName,
		AutoWait: autoWait,
		AutoConnect: func(context.Context) error {
			if runtime.GOOS == "windows" {
				// GUI automation opens FortiClient, selects the tunnel,
				// fills credentials and clicks Connect.
				return tryGUIConnect(cfg.TunnelName, cfg.Username, password, allowInsecureCert)
			}
			return connectV6(cfg.TunnelName, cfg.Host, cfg.Port, cfg.CertPath, cfg.KeyPath, cfg.Username, password)
		},
		DisconnectWait: 2 * time.Second,
	}
	p.Disconnect = func(ctx context.Context) error { return a.disconnectAutomated(ctx, p.Matcher, cfg.TunnelName) }
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

// disconnectAutomated tries GUI automation (Windows) and the CLI. An error
// means the tunnel is still up; attach mode then shows the translated "disconnect
// in FortiClient" guidance and waits. Force-disabling the network adapter
// was dropped: it needs administrator rights, and the privileged helper
// only runs a closed list of operations (no "disable any adapter" op).
func (a *Adapter) disconnectAutomated(ctx context.Context, m appattach.Matcher, tunnelName string) error {
	_ = tryGUIDisconnect(tunnelName)
	_ = disconnectV6(tunnelName)
	if a.sess.WaitDown(ctx, m, disconnectGrace) == nil {
		return nil
	}
	if _, ok := a.sess.Detect(m); !ok {
		return nil
	}
	return errors.New("forticlient: GUI and CLI disconnect did not drop the tunnel")
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
