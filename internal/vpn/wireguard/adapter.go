// Package wireguard implements the VPNAdapter interface for WireGuard.
// Uses wg-quick on Linux/macOS and wireguard.exe on Windows, run through
// internal/privops (the Kongtrol helper when not elevated).
// The interface name is derived from the config filename: wg0.conf → wg0.
//
// Note: wg-quick may handle DNS internally on some systems (via resolvconf/systemd-resolved).
// Kongtrol's DNS guard is the authoritative DNS setter — set DNS in the .conf file
// and the adapter will read it and pass it to the DNS guard via TunnelInfo().DNS.
package wireguard

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("wireguard", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for WireGuard.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	ifaceName   string
	assignedIP  net.IP
	dnsServers  []net.IP
	connectedAt time.Time
	lastCfg     vpn.AdapterConfig
	// profileConfig is the profile .conf (never a temporary patched copy),
	// sent on disconnect so the helper can adopt a tunnel it did not start.
	profileConfig string

	// Liveness (ADR-0030): handshake + counters read through privops
	// at most every statsEvery; statsMu keeps one read at a time.
	statsMu  sync.Mutex
	statsAt  time.Time
	statsGen uint64 // bumped on reset: drops a read of the old tunnel
	tracker  healthTracker
	health   *vpn.TunnelHealth
}

// statsEvery is how often Status reads the tunnel counters (each read is
// one helper call running wg show).
const statsEvery = 10 * time.Second

// now is the clock (tests).
var now = time.Now

// Configure pre-seeds the adapter config so Status() can probe the WireGuard
// interface even when Connect() has not been called in this process lifetime.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cfg.ConfigPath != "" && !isPatchedCopy(cfg.ConfigPath) {
		a.profileConfig = cfg.ConfigPath
	}
	if a.lastCfg.ConfigPath == "" {
		a.lastCfg = cfg
	}
	if a.ifaceName == "" && cfg.ConfigPath != "" {
		a.ifaceName = interfaceFromConfig(cfg.ConfigPath)
		if cfg.TunnelName != "" {
			a.ifaceName = cfg.TunnelName
		}
	}
}

func (a *Adapter) Name() string { return "wireguard" }
func (a *Adapter) Version() string {
	out, err := runCmd("wg", "--version")
	if err == nil {
		return out
	}
	return "unknown"
}

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: true, // WireGuard AllowedIPs is split-tunnel by design
		SupportsMultiConn:   true, // multiple wg interfaces can coexist
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("wireguard: already %s", a.status)
	}
	// A tunnel that is up but stopped responding is replaced, not reused.
	restart := a.health != nil && a.health.State == vpn.HealthNoResponse

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg
	if cfg.ConfigPath != "" && !isPatchedCopy(cfg.ConfigPath) {
		a.profileConfig = cfg.ConfigPath
	}

	// WireGuard uses key-based auth — no password needed.
	cfg.Password = ""

	configPath := cfg.ConfigPath
	if configPath == "" {
		a.status = vpn.StatusError
		return fmt.Errorf("wireguard: ConfigPath is required (path to .conf file)")
	}

	// Derive interface name and parse config metadata before bringing up.
	a.ifaceName = interfaceFromConfig(configPath)
	if cfg.TunnelName != "" {
		a.ifaceName = cfg.TunnelName
	}

	// Pre-read assigned IP and DNS from the config file so TunnelInfo() is
	// populated even before the first handshake.
	a.assignedIP, _ = parseConfigAddress(configPath)
	a.dnsServers = parseConfigDNS(configPath)

	if err := wgUp(configPath, a.ifaceName, restart, cfg.DefaultEgress); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("wireguard: %w", err)
	}

	// Wait for the network interface to appear.
	if err := waitForInterface(a.ifaceName, 15*time.Second); err != nil {
		a.status = vpn.StatusError
		return err
	}

	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	a.resetHealthLocked()
	return nil
}

// resetHealthLocked starts judging the tunnel afresh. Call with a.mu held.
func (a *Adapter) resetHealthLocked() {
	a.tracker.reset(now())
	a.statsAt = time.Time{}
	a.statsGen++
	a.health = nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := wgDown(a.ifaceName, a.profileConfig); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("wireguard: disconnect: %w", err)
	}

	a.status = vpn.StatusDisconnected
	a.resetHealthLocked()
	return nil
}

func (a *Adapter) Reconnect(ctx context.Context) error {
	if err := a.Disconnect(ctx); err != nil {
		return err
	}
	return a.Connect(ctx, a.lastCfg)
}

func (a *Adapter) Status() vpn.Status {
	a.mu.Lock()
	iface := a.ifaceName
	if iface == "" {
		// No iface known in-memory — nothing to check.
		defer a.mu.Unlock()
		return a.status.Normalize()
	}
	if !ifaceExists(iface) {
		defer a.mu.Unlock()
		a.health = nil
		// Interface not in net.Interfaces(). On Windows, the service may be
		// starting or restarting — avoid false error during that window.
		if tunnelServiceRunning(iface) {
			return vpn.StatusConnecting
		}
		if a.status == vpn.StatusConnected {
			a.status = vpn.StatusError
		}
		return a.status.Normalize()
	}

	// Interface is up — reconcile in-memory state, unless it is up but its
	// server stopped answering (kept as StatusError until it answers).
	if a.status != vpn.StatusConnected && !a.noResponseLocked() {
		a.status = vpn.StatusConnected
		a.resetHealthLocked()
		if a.connectedAt.IsZero() {
			a.connectedAt = time.Now()
		}
		if a.assignedIP == nil {
			a.assignedIP, _ = parseConfigAddress(a.lastCfg.ConfigPath)
		}
		if len(a.dnsServers) == 0 {
			a.dnsServers = parseConfigDNS(a.lastCfg.ConfigPath)
		}
	}
	profileCfg := a.profileConfig
	a.mu.Unlock()

	// The interface existing is not enough: WireGuard has no connection
	// state, so judge it by the peer's latest handshake (ADR-0030).
	a.refreshHealth(iface, profileCfg)

	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case a.ifaceName != iface:
		// Reconfigured meanwhile: report what we have.
	case a.noResponseLocked() && a.status == vpn.StatusConnected:
		a.status = vpn.StatusError
	case !a.noResponseLocked() && a.status == vpn.StatusError:
		a.status = vpn.StatusConnected // answering again
	}
	return a.status.Normalize()
}

// noResponseLocked reports whether the tunnel is up but its server stopped
// answering. Call with a.mu held.
func (a *Adapter) noResponseLocked() bool {
	return a.health != nil && a.health.State == vpn.HealthNoResponse
}

// refreshHealth reads the tunnel counters through privops at most every
// statsEvery and updates a.health. A concurrent caller skips the read.
func (a *Adapter) refreshHealth(iface, profileCfg string) {
	if !a.statsMu.TryLock() {
		return
	}
	defer a.statsMu.Unlock()
	a.mu.Lock()
	due := a.statsAt.IsZero() || now().Sub(a.statsAt) >= statsEvery
	gen := a.statsGen
	a.mu.Unlock()
	if !due {
		return
	}
	st, err := wgStatsFunc(iface, profileCfg)
	a.mu.Lock()
	defer a.mu.Unlock()
	if gen != a.statsGen || a.ifaceName != iface {
		return // the tunnel was re-created meanwhile: this read is stale
	}
	a.statsAt = now()
	h := a.tracker.observe(a.statsAt, st, err)
	a.health = &h
}

// TunnelHealth reports whether the connected tunnel's server answers
// (vpn.HealthReporter). nil while the tunnel is down or not read yet.
func (a *Adapter) TunnelHealth() *vpn.TunnelHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.health == nil || (a.status != vpn.StatusConnected && a.status != vpn.StatusError) {
		return nil
	}
	h := *a.health
	return &h
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != vpn.StatusConnected {
		return nil, nil
	}

	info := &vpn.TunnelInfo{
		InterfaceName: a.ifaceName,
		AssignedIP:    a.assignedIP,
		DNS:           a.dnsServers,
		ConnectedAt:   a.connectedAt,
	}

	// Byte counts from the last counters read (Status); without them the
	// collector uses the OS interface counters.
	if rx, tx, ok := a.tracker.totals(); ok {
		info.BytesSent, info.BytesReceived = tx, rx
	}

	// Resolve actual interface IP in case wg-quick adjusted it. Don't assume
	// addrs[0] is IPv4 — on Windows the WireGuard TUN adapter commonly lists
	// a link-local IPv6 address first, and To4() on that silently wipes out
	// the IPv4 already parsed from the config.
	if iface, err := net.InterfaceByName(a.ifaceName); err == nil {
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				if v4 := ipnet.IP.To4(); v4 != nil {
					info.AssignedIP = v4
					break
				}
			}
		}
	}

	return info, nil
}
