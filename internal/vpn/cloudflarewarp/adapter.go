// Package cloudflarewarp implements the VPNAdapter interface for Cloudflare WARP.
// Requires the Cloudflare WARP client to be installed and registered.
// Run `warp-cli register` once before first use.
//
// DNS note: WARP hardcodes 1.1.1.1 / 1.0.0.1 as DNS servers; the adapter
// surfaces these in TunnelInfo so the DNS guard can validate them.
package cloudflarewarp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("cloudflarewarp", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for Cloudflare WARP.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	iface       string
	assignedIP  net.IP
	connectedAt time.Time
}

func (a *Adapter) Name() string    { return "cloudflarewarp" }
func (a *Adapter) Version() string { return detectVersion() }

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: false, // WARP routes all traffic by default
		SupportsMultiConn:   false, // single WARP session per device
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("cloudflarewarp: already %s", a.status)
	}

	a.status = vpn.StatusConnecting

	if err := warpConnect(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("cloudflarewarp: connect: %w", err)
	}

	// Poll until WARP reports "connected" or context cancels.
	timeout := 30 * time.Second
	done := make(chan error, 1)
	go func() { done <- waitConnected(timeout) }()

	select {
	case <-ctx.Done():
		a.status = vpn.StatusError
		return ctx.Err()
	case err := <-done:
		if err != nil {
			a.status = vpn.StatusError
			return err
		}
	}

	iface, ip := warpInterface()
	a.iface = iface
	a.assignedIP = ip
	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := warpDisconnect(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("cloudflarewarp: disconnect: %w", err)
	}

	a.iface = ""
	a.assignedIP = nil
	a.status = vpn.StatusDisconnected
	return nil
}

func (a *Adapter) Reconnect(ctx context.Context) error {
	if err := a.Disconnect(ctx); err != nil {
		return err
	}
	// lastCfg not needed — WARP has no per-connect credentials.
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status = vpn.StatusConnecting
	if err := warpConnect(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("cloudflarewarp: reconnect: %w", err)
	}
	if err := waitConnected(30 * time.Second); err != nil {
		a.status = vpn.StatusError
		return err
	}
	iface, ip := warpInterface()
	a.iface = iface
	a.assignedIP = ip
	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Status() vpn.Status {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != vpn.StatusConnected {
		return a.status.Normalize()
	}

	raw, err := warpStatusOutput()
	if err != nil {
		return vpn.StatusError
	}
	switch parseWarpStatus(raw) {
	case "connected":
		return vpn.StatusConnected
	case "connecting":
		return vpn.StatusConnecting
	default:
		return vpn.StatusDisconnected
	}
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status != vpn.StatusConnected {
		return nil, nil
	}

	// Refresh interface/IP in case it changed after connect.
	if iface, ip := warpInterface(); iface != "" {
		a.iface = iface
		a.assignedIP = ip
	}

	return &vpn.TunnelInfo{
		InterfaceName: a.iface,
		AssignedIP:    a.assignedIP,
		DNS:           warpDNS,
		ConnectedAt:   a.connectedAt,
	}, nil
}
