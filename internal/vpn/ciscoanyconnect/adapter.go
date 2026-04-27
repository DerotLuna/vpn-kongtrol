// Package ciscoanyconnect implements the VPNAdapter interface for Cisco AnyConnect
// and Cisco Secure Client (formerly AnyConnect). Uses the vpncli CLI binary.
//
// Prerequisites:
//   - Cisco AnyConnect / Secure Client installed
//   - VPN agent service (vpnagentd) must be running
//   - On Windows: administrator privileges required
package ciscoanyconnect

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("ciscoanyconnect", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for Cisco AnyConnect / Secure Client.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	assignedIP  net.IP
	dnsServers  []net.IP
	connectedAt time.Time
	lastCfg     vpn.AdapterConfig
}

func (a *Adapter) Name() string    { return "ciscoanyconnect" }
func (a *Adapter) Version() string { return detectVersion() }

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: false, // AnyConnect manages its own routing
		SupportsMultiConn:   false, // exclusive — only one AnyConnect tunnel at a time
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("ciscoanyconnect: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	// Verify the agent is running before attempting to connect.
	if err := checkServiceRunning(); err != nil {
		a.status = vpn.StatusError
		return err
	}

	if err := connect(cfg.Host, cfg.Username, cfg.Password); err != nil {
		a.status = vpn.StatusError
		cfg.Password = ""
		return fmt.Errorf("ciscoanyconnect: connect: %w", err)
	}
	cfg.Password = ""

	// Wait for Connected state.
	if err := a.waitConnected(ctx); err != nil {
		a.status = vpn.StatusError
		return err
	}

	// Collect tunnel info.
	if raw, err := statsOutput(); err == nil {
		a.dnsServers = parseDNS(raw)
	}
	if raw, err := statusOutput(); err == nil {
		_, ipStr := parseStatus(raw)
		a.assignedIP = net.ParseIP(ipStr)
	}

	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := disconnect(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("ciscoanyconnect: disconnect: %w", err)
	}

	a.assignedIP = nil
	a.dnsServers = nil
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

	return &vpn.TunnelInfo{
		AssignedIP:  a.assignedIP,
		DNS:         a.dnsServers,
		ConnectedAt: a.connectedAt,
	}, nil
}

func (a *Adapter) waitConnected(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for AnyConnect to report Connected state")
		case <-ticker.C:
			raw, err := statusOutput()
			if err != nil {
				continue
			}
			connected, _ := parseStatus(raw)
			if connected {
				return nil
			}
		}
	}
}
