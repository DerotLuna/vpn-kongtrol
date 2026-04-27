// Package globalprotect implements the VPNAdapter interface for Palo Alto GlobalProtect.
// Supported on Windows and macOS only (no official Linux client).
//
// SSO note: if the portal is configured for SSO-only authentication,
// Connect() returns ErrSSORequired — connect manually via the GlobalProtect
// client, then Kongtrol will detect the tunnel and manage routes on top of it.
package globalprotect

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func init() {
	vpn.Register("globalprotect", func() vpn.VPNAdapter { return &Adapter{} })
}

// Adapter implements vpn.VPNAdapter for Palo Alto GlobalProtect.
type Adapter struct {
	mu          sync.RWMutex
	status      vpn.Status
	ifaceName   string
	assignedIP  net.IP
	connectedAt time.Time
	lastCfg     vpn.AdapterConfig
}

func (a *Adapter) Name() string    { return "globalprotect" }
func (a *Adapter) Version() string { return detectVersion() }

func (a *Adapter) Capabilities() vpn.Capabilities {
	return vpn.Capabilities{
		SupportsSplitTunnel: false, // GlobalProtect manages its own routing policy
		SupportsMultiConn:   false, // exclusive tunnel
		SupportsReconnect:   true,
	}
}

func (a *Adapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, err := binaryPath(); err != nil {
		a.status = vpn.StatusError
		return err
	}
	if cfg.Host == "" {
		return fmt.Errorf("globalprotect: Host (portal address) is required")
	}
	if a.status == vpn.StatusConnected || a.status == vpn.StatusConnecting {
		return fmt.Errorf("globalprotect: already %s", a.status)
	}

	a.status = vpn.StatusConnecting
	a.lastCfg = cfg

	err := connect(cfg.Host, cfg.Username, cfg.Password)
	cfg.Password = ""

	if err != nil {
		// SSO-required: try passive detection (user connected manually).
		if err == ErrSSORequired {
			iface, ip, detectErr := detectInterface(5 * time.Second)
			if detectErr == nil {
				a.ifaceName = iface
				a.assignedIP = ip
				a.connectedAt = time.Now()
				a.status = vpn.StatusConnected
				return nil
			}
		}
		a.status = vpn.StatusError
		return err
	}

	// Wait for tunnel interface.
	iface, ip, err := detectInterface(20 * time.Second)
	if err != nil {
		a.status = vpn.StatusError
		return err
	}

	a.ifaceName = iface
	a.assignedIP = ip
	a.connectedAt = time.Now()
	a.status = vpn.StatusConnected
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := disconnect(); err != nil {
		a.status = vpn.StatusError
		return err
	}

	a.ifaceName = ""
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

	return &vpn.TunnelInfo{
		InterfaceName: a.ifaceName,
		AssignedIP:    a.assignedIP,
		ConnectedAt:   a.connectedAt,
		// GlobalProtect DNS is set by the PANGPS service — not exposed via CLI.
		// DNS guard will use FallbackDNS from config.
	}, nil
}
