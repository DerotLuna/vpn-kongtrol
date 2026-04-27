// Package openvpn implements the VPNAdapter interface for OpenVPN.
// It supports multiple simultaneous instances (one per config file),
// using dynamic management port allocation to avoid conflicts.
package openvpn

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
	vpn.Register("openvpn", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for OpenVPN.
type Adapter struct {
	mu         sync.RWMutex
	status     vpn.Status
	proc       *process
	lastCfg    vpn.AdapterConfig
	tunnelInfo *vpn.TunnelInfo
}

func (a *Adapter) Name() string    { return "openvpn" }
func (a *Adapter) Version() string { return detectVersion() }

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: true,
		SupportsMultiConn:   true,
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("openvpn: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	p, err := start(cfg.ConfigPath, cfg.CertPath, cfg.KeyPath, cfg.Username, cfg.Password)
	// Zero the password immediately after use.
	cfg.Password = ""

	if err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("openvpn: connect: %w", err)
	}

	a.proc = p

	// Wait for CONNECTED state, respecting ctx.
	if err := a.waitConnected(ctx); err != nil {
		_ = p.stop()
		a.proc = nil
		a.status = vpn.StatusError
		return fmt.Errorf("openvpn: waiting for connection: %w", err)
	}

	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.proc == nil {
		a.status = vpn.StatusDisconnected
		return nil
	}

	if err := a.proc.stop(); err != nil {
		a.status = vpn.StatusError
		return fmt.Errorf("openvpn: disconnect: %w", err)
	}

	a.proc = nil
	a.tunnelInfo = nil
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

	if a.proc == nil || a.proc.mgmt == nil {
		return nil, nil
	}

	stateStr, err := a.proc.mgmt.State()
	if err != nil {
		return nil, fmt.Errorf("openvpn: get state: %w", err)
	}

	state, localIPStr := parseState(stateStr)
	if state != "CONNECTED" {
		return nil, nil
	}

	info := &vpn.TunnelInfo{
		ConnectedAt: time.Now(), // TODO: parse from state timestamp
	}
	if localIPStr != "" {
		info.AssignedIP = net.ParseIP(localIPStr)
	}
	return info, nil
}

// waitConnected polls the management interface until CONNECTED or ctx is done.
func (a *Adapter) waitConnected(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for CONNECTED state")
		case <-ticker.C:
			stateStr, err := a.proc.mgmt.State()
			if err != nil {
				continue
			}
			state, _ := parseState(stateStr)
			switch strings.ToUpper(state) {
			case "CONNECTED":
				return nil
			case "AUTH_FAILED":
				return fmt.Errorf("authentication failed")
			case "EXITING":
				return fmt.Errorf("process exited unexpectedly")
			}
		}
	}
}

// detectVersion runs `openvpn --version` and returns the version string.
func detectVersion() string {
	// openvpn --version exits with code 1 but prints the version to stdout.
	// We capture the output regardless of exit code.
	out, _ := runCmd("openvpn", "--version")
	lines := strings.SplitN(out, "\n", 2)
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "unknown"
}
