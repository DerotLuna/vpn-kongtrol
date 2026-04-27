// Package protonvpn implements the VPNAdapter interface for ProtonVPN.
// Requires protonvpn-cli v3.x installed and pre-authenticated.
// Run `protonvpn-cli login` once before using Kongtrol.
package protonvpn

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
	vpn.Register("protonvpn", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for ProtonVPN CLI.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	connectedAt time.Time
	lastCfg     vpn.AdapterConfig
}

func (a *Adapter) Name() string { return "protonvpn" }

func (a *Adapter) Version() string {
	out, _ := cliOutput("--version")
	lines := strings.SplitN(strings.TrimSpace(out), "\n", 2)
	if len(lines) > 0 {
		return lines[0]
	}
	return "unknown"
}

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: true,
		SupportsMultiConn:   false,
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("protonvpn: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	if err := connect(cfg.Extra["server"], cfg.Extra["protocol"]); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("protonvpn: connect: %w", err)
	}

	// Wait for the connection to be confirmed.
	if err := a.waitConnected(ctx); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("protonvpn: waiting for connection: %w", err)
	}

	a.status = vpn.StatusConnected
	a.connectedAt = time.Now()
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := disconnect(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("protonvpn: disconnect: %w", err)
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

	raw, err := statusOutput()
	if err != nil {
		return nil, fmt.Errorf("protonvpn: status: %w", err)
	}

	connected, serverIP, assignedIP := parseStatus(raw)
	if !connected {
		return nil, nil
	}

	info := &vpn.TunnelInfo{
		ConnectedAt: a.connectedAt,
	}
	if assignedIP != "" {
		info.AssignedIP = net.ParseIP(assignedIP)
	}
	if serverIP != "" {
		info.RemoteIP = net.ParseIP(serverIP)
	}
	return info, nil
}

// waitConnected polls protonvpn-cli status until connected or ctx expires.
func (a *Adapter) waitConnected(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for ProtonVPN to connect")
		case <-ticker.C:
			raw, err := statusOutput()
			if err != nil {
				continue
			}
			connected, _, _ := parseStatus(raw)
			if connected {
				return nil
			}
		}
	}
}
