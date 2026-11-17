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
	// connectedAt is when the tunnel last reached CONNECTED.
	connectedAt time.Time
}

func (a *Adapter) Name() string    { return "openvpn" }
func (a *Adapter) Version() string { return a.detectVersion() }

// Configure pre-seeds the profile config (binary_path for Version(), etc.)
// before the first Connect.
func (a *Adapter) Configure(cfg vpn.AdapterConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg.Password = ""
	a.lastCfg = cfg
}

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: true,
		SupportsMultiConn:   true,
		SupportsReconnect:   true,
	}
}

// reapExitedLocked clears an openvpn that exited on its own (killed,
// crashed, fatal error) while the adapter still recorded it as connected:
// it frees the dead process/helper session and leaves the adapter in
// StatusError, so Connect can start a fresh one instead of answering
// "already connected" forever. Caller holds a.mu.
func (a *Adapter) reapExitedLocked() {
	if a.proc == nil || !a.proc.hasExited() {
		return
	}
	a.proc.release()
	a.proc = nil
	a.tunnelInfo = nil
	a.connectedAt = time.Time{}
	if a.status == vpn.StatusConnected {
		a.status = vpn.StatusError
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.reapExitedLocked()
	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("openvpn: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	if cfg.ConfigPath == "" {
		a.status = vpn.StatusError
		return fmt.Errorf("openvpn: ConfigPath is required (path to .ovpn/.conf file)")
	}

	// Elevated: spawn openvpn ourselves. Otherwise the helper runs its own
	// trusted openvpn binary and binary_path is not used.
	var bin string
	if isElevated() {
		var err error
		if bin, err = FindBinary(cfg.Extra["binary_path"]); err != nil {
			cfg.Password = ""
			a.lastCfg.Password = ""
			a.status = vpn.StatusError
			return fmt.Errorf("openvpn: %w", err)
		}
	}

	p, err := start(ctx, startOptions{
		Binary:     bin,
		ConfigPath: cfg.ConfigPath,
		CertPath:   cfg.CertPath,
		KeyPath:    cfg.KeyPath,
	}, cfg.Username, cfg.Password)
	// Zero the password immediately after use.
	cfg.Password = ""
	a.lastCfg.Password = ""

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

	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.reapExitedLocked()
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
	// A tunnel whose openvpn process died (killed, crashed, fatal error)
	// must not keep reporting connected — the watchdog relies on this.
	if a.status == vpn.StatusConnected && a.proc != nil && a.proc.hasExited() {
		return vpn.StatusError
	}
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

	iface, dns := a.proc.log.snapshot()
	info := &vpn.TunnelInfo{
		InterfaceName: iface,
		DNS:           dns,
		ConnectedAt:   a.connectedAt,
	}
	if localIPStr != "" {
		info.AssignedIP = net.ParseIP(localIPStr)
	}
	// Live traffic from the management interface. On failure the collector
	// falls back to the OS interface counters.
	if sent, recv, err := a.proc.mgmt.ByteTotals(); err == nil {
		info.BytesSent, info.BytesReceived = sent, recv
	}
	if info.InterfaceName == "" {
		// The device line was not seen (e.g. it scrolled out of the
		// helper's log tail): find the interface holding the tunnel IP.
		info.InterfaceName = interfaceWithIP(info.AssignedIP)
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
			if a.proc.hasExited() {
				return a.proc.exitError()
			}
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
func (a *Adapter) detectVersion() string {
	a.mu.RLock()
	override := a.lastCfg.Extra["binary_path"]
	a.mu.RUnlock()
	bin, err := FindBinary(override)
	if err != nil {
		return "unknown"
	}
	// openvpn --version exits with code 1 but prints the version to stdout.
	// We capture the output regardless of exit code.
	out, _ := runCmd(bin, "--version")
	lines := strings.SplitN(out, "\n", 2)
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "unknown"
}

// TunnelRoutes reports the routes the server pushed (PUSH_REPLY route /
// route-ipv6 / redirect-gateway). Nil when not connected or nothing was
// pushed.
func (a *Adapter) TunnelRoutes() *vpn.TunnelRoutes {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.proc == nil || a.status != vpn.StatusConnected {
		return nil
	}
	routes, redirect, ok := a.proc.log.pushed()
	if !ok {
		return nil
	}
	return &vpn.TunnelRoutes{Origin: vpn.OriginServer, Routes: routes, AllTraffic: redirect}
}
