// Package tailscale implements the VPNAdapter interface for Tailscale.
// Requires the Tailscale daemon (tailscaled) to be running and the device
// to be already authenticated (run `tailscale login` once).
//
// Auth key note: if cfg.Password contains a Tailscale auth key, it is passed
// as --authkey on connect. Leave it empty to reuse the existing session.
//
// Disconnect note: `tailscale down` pauses routing but does NOT log out.
// The daemon reconnects on the next `tailscale up` or daemon restart.
// To fully de-authenticate, run `tailscale logout` manually.
package tailscale

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("tailscale", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for Tailscale.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	assignedIP  net.IP
	connectedAt time.Time
	lastCfg     vpn.AdapterConfig
}

func (a *Adapter) Name() string    { return "tailscale" }
func (a *Adapter) Version() string { return detectVersion() }

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: true,  // Tailscale supports exit nodes + subnet routing
		SupportsMultiConn:   false, // single Tailscale network per device
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("tailscale: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	// cfg.Password = auth key (optional); cfg.Host = exit node (optional).
	authKey := cfg.Password
	cfg.Password = ""
	exitNode := cfg.Host

	if err := tsUp(authKey, exitNode); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("tailscale: connect: %w", err)
	}

	// Wait for Running state.
	if err := a.waitRunning(ctx); err != nil {
		a.status = vpn.StatusError
		return err
	}

	s, _ := tsStatus()
	if s != nil {
		a.assignedIP = parseAssignedIP(s)
	}

	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := tsDown(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("tailscale: disconnect: %w", err)
	}

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

	if a.status != vpn.StatusConnected {
		return a.status.Normalize()
	}

	s, err := tsStatus()
	if err != nil || s == nil {
		return vpn.StatusError
	}
	switch s.BackendState {
	case "Running":
		return vpn.StatusConnected
	case "Connecting":
		return vpn.StatusConnecting
	default:
		return vpn.StatusDisconnected
	}
}

func (a *Adapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != vpn.StatusConnected {
		return nil, nil
	}

	info := &vpn.TunnelInfo{
		AssignedIP:  a.assignedIP,
		ConnectedAt: a.connectedAt,
		// Tailscale DNS is managed by the MagicDNS system in tailscaled.
		// Do not set DNS here — let the daemon own it.
	}

	// Refresh IP from live status.
	if s, err := tsStatus(); err == nil && s != nil {
		info.AssignedIP = parseAssignedIP(s)
	}

	return info, nil
}

func (a *Adapter) waitRunning(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("tailscale: timeout waiting for Running state")
		case <-ticker.C:
			s, err := tsStatus()
			if err != nil {
				continue
			}
			if s.BackendState == "Running" {
				return nil
			}
			if s.BackendState == "NeedsLogin" {
				return fmt.Errorf("tailscale: device not authenticated — run 'tailscale login'")
			}
		}
	}
}
