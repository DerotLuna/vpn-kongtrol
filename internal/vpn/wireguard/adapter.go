// Package wireguard implements the VPNAdapter interface for WireGuard.
// Uses wg-quick on Linux/macOS and wireguard.exe on Windows.
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
}

// Configure pre-seeds the adapter config so Status() can probe the WireGuard
// interface even when Connect() has not been called in this process lifetime.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
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
		SupportsSplitTunnel: true,  // WireGuard AllowedIPs is split-tunnel by design
		SupportsMultiConn:   true,  // multiple wg interfaces can coexist
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("wireguard: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

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

	if err := wgUp(configPath); err != nil {
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
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := wgDown(a.ifaceName); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("wireguard: disconnect: %w", err)
	}

	a.status = vpn.StatusDisconnected
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
	defer a.mu.Unlock()

	iface := a.ifaceName
	if iface == "" {
		// No iface known in-memory — nothing to check.
		return a.status.Normalize()
	}

	raw, err := wgShow(iface)
	if err != nil || raw == "" {
		// wg show failed — wg CLI may be unavailable or the named pipe not yet ready.
		// Fall back to checking the network interface directly.
		if ifaceExists(iface) {
			// Interface is up; reconcile state as connected below.
			raw = "fallback"
		} else {
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
	}

	// Interface is up — reconcile in-memory state.
	if a.status != vpn.StatusConnected {
		a.status = vpn.StatusConnected
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
	return vpn.StatusConnected
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

	// Live byte counts from wg transfer.
	if raw, err := wgTransfer(a.ifaceName); err == nil {
		info.BytesSent, info.BytesReceived = parseTransfer(raw)
	}

	// Resolve actual interface IP in case wg-quick adjusted it.
	if iface, err := net.InterfaceByName(a.ifaceName); err == nil {
		if addrs, err := iface.Addrs(); err == nil && len(addrs) > 0 {
			if ipnet, ok := addrs[0].(*net.IPNet); ok {
				info.AssignedIP = ipnet.IP.To4()
			}
		}
	}

	return info, nil
}
