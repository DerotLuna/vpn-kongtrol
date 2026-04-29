// Package forticlient implements the VPNAdapter interface for FortiClient.
// Supports FortiClient 6.4.x via CLI (Windows: /vpnconnect, Linux/macOS: forticlientsslvpn).
// If the enterprise EMS policy blocks CLI control, the adapter detects the tunnel
// passively and only manages routes on top of it (ErrCLIBlocked fallback mode).
package forticlient

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("forticlient", func() vpn.VPNAdapter { return &Adapter{} })
}

// ErrCLIBlocked is returned when FortiClient ignores CLI connect commands,
// typically because an EMS policy disables CLI control.
var ErrCLIBlocked = fmt.Errorf("forticlient: CLI connect blocked by EMS policy — connect manually then run 'kongtrol up'")

// Adapter implements vpn.VPNAdapter for FortiClient.
type Adapter struct {
	mu            sync.RWMutex
	status        vpn.Status
	tunnelIface   string
	assignedIP    net.IP
	connectedAt   time.Time
	lastCfg       vpn.AdapterConfig
	detectedMajor int
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

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: false, // FortiClient manages its own routing
		SupportsMultiConn:   false, // FortiClient is exclusive by design
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("forticlient: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	// Attempt CLI connect (v6.4.x).
	err := connectV6(cfg.TunnelName, cfg.Host, cfg.Port, cfg.CertPath, cfg.KeyPath, cfg.Username, cfg.Password)
	cfg.Password = "" // zero immediately

	if err != nil {
		// CLI launch failed outright — try passive detection in case tunnel is already up.
		iface, ip, detectErr := detectTunnelInterface(5 * time.Second)
		if detectErr != nil {
			a.status = vpn.StatusError
			return ErrCLIBlocked
		}
		a.tunnelIface = iface
		a.assignedIP = ip
		a.connectedAt = time.Now()
		a.status = vpn.StatusConnected
		return nil
	}

	// CLI launched — wait for tunnel interface to appear.
	// On Windows, FortiClient runs in background and may require GUI interaction
	// (accept cert, MFA, etc.), so we allow a generous timeout.
	iface, ip, err := detectTunnelInterface(60 * time.Second)
	if err != nil {
		// Tunnel didn't appear — FortiClient GUI may need manual action.
		// Reset to disconnected (not error) so a retry works cleanly.
		a.status = vpn.StatusDisconnected
		return fmt.Errorf("forticlient: tunnel did not come up within 60s — connect manually in the FortiClient GUI, then run 'kongtrol up' again")
	}

	a.tunnelIface = iface
	a.assignedIP = ip
	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := disconnectV6(a.lastCfg.TunnelName); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("forticlient: disconnect: %w", err)
	}

	a.tunnelIface = ""
	a.assignedIP = nil
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
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status.Normalize()
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != vpn.StatusConnected {
		return nil, nil
	}

	info := &vpn.TunnelInfo{
		InterfaceName: a.tunnelIface,
		AssignedIP:    a.assignedIP,
		ConnectedAt:   a.connectedAt,
	}

	// Get current byte counts from the interface.
	if a.tunnelIface != "" {
		if iface, err := net.InterfaceByName(a.tunnelIface); err == nil {
			_ = iface // byte counts are OS-specific; delegated to monitor package
		}
	}

	return info, nil
}
