// Package forticlient implements the VPNAdapter interface for FortiClient.
// Supports FortiClient 6.4.x via CLI (Windows: /vpnconnect, Linux/macOS: forticlientsslvpn).
// If the enterprise EMS policy blocks CLI control, the adapter detects the tunnel
// passively and only manages routes on top of it (ErrCLIBlocked fallback mode).
package forticlient

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
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

	// ── Step 0: fast-path — tunnel already up (user connected manually) ─────
	// Check before doing anything so we don't interfere with an active session.
	if iface, ip, err := detectTunnelInterface(ctx, 4*time.Second); err == nil {
		a.tunnelIface = iface
		a.assignedIP = ip
		a.connectedAt = time.Now()
		a.status = vpn.StatusConnected
		cfg.Password = ""
		return nil
	}
	if ctx.Err() != nil {
		cfg.Password = ""
		return ctx.Err()
	}

	// ── Step 1: trigger connection ────────────────────────────────────────────
	// On Windows: run GUI automation in a goroutine so it doesn't block
	// detectTunnelInterface — Ctrl+C can interrupt the detection loop even
	// while the PS script is still running.
	// On other OSes: fall back to CLI.
	tunnelName := cfg.TunnelName
	username := cfg.Username
	password := cfg.Password
	host := cfg.Host
	port := cfg.Port
	certPath := cfg.CertPath
	keyPath := cfg.KeyPath
	cfg.Password = "" // zero immediately

	if runtime.GOOS == "windows" {
		// Launch GUI automation asynchronously — it opens FortiClient, fills
		// credentials, clicks Connect. Detection loop below waits for the result.
		go func() {
			if err := tryGUIConnect(tunnelName, username, password); err != nil {
				fmt.Fprintf(os.Stderr, "forticlient gui: %v\n", err)
			}
		}()
	} else {
		// CLI path for Linux/macOS.
		if err := connectV6(tunnelName, host, port, certPath, keyPath, username, password); err != nil {
			// CLI failed — check if tunnel already up (manual connection).
			iface, ip, detectErr := detectTunnelInterface(ctx, 5*time.Second)
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
	}

	// ── Step 2: wait for tunnel interface to appear ───────────────────────────
	// Context-aware: Ctrl+C cancels immediately instead of waiting 90s.
	iface, ip, err := detectTunnelInterface(ctx, 90*time.Second)
	if err != nil {
		a.status = vpn.StatusDisconnected
		if ctx.Err() != nil {
			return ctx.Err() // user cancelled
		}
		if runtime.GOOS == "windows" {
			return fmt.Errorf("forticlient: tunnel did not come up within 90s — check FortiClient window for errors or connect manually")
		}
		return fmt.Errorf("forticlient: tunnel did not come up within 90s — connect manually in FortiClient, then run 'kongtrol up' again")
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

	iface := a.tunnelIface

	// Try GUI automation first (Windows) — clicks the Disconnect button in the GUI.
	_ = tryGUIDisconnect(a.lastCfg.TunnelName)

	// Also try CLI disconnect as belt-and-suspenders.
	_ = disconnectV6(a.lastCfg.TunnelName)

	// Wait up to 8s for the tunnel to go down.
	if iface != "" && fortiIfaceUp(iface) {
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)
			if !fortiIfaceUp(iface) {
				break
			}
		}
	}

	// Last resort: force-disable the network adapter.
	if iface != "" && fortiIfaceUp(iface) {
		if err := disableFortiAdapter(iface); err != nil {
			a.status = vpn.StatusError
			return fmt.Errorf("forticlient: disconnect: GUI, CLI, and adapter disable all failed: %w", err)
		}
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
	a.mu.Lock()
	defer a.mu.Unlock()

	// If we think we're connected, verify the tunnel interface is still up.
	if a.status == vpn.StatusConnected && a.tunnelIface != "" {
		if !fortiIfaceUp(a.tunnelIface) {
			a.status = vpn.StatusDisconnected
			a.tunnelIface = ""
			a.assignedIP = nil
		}
	}

	// If we think we're disconnected, check if the tunnel appeared externally
	// (user connected manually in FortiClient GUI).
	if a.status == vpn.StatusDisconnected {
		if iface, ip := probeFortiInterface(); iface != "" {
			a.tunnelIface = iface
			a.assignedIP = ip
			a.connectedAt = time.Now()
			a.status = vpn.StatusConnected
		}
	}

	return a.status.Normalize()
}

// fortiIfaceUp checks whether the named interface exists and has an IPv4 address.
func fortiIfaceUp(name string) bool {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return false
	}
	// Check that interface is flagged up.
	if iface.Flags&net.FlagUp == 0 {
		return false
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			return true
		}
	}
	return false
}

// probeFortiInterface is a lightweight check (no PowerShell) that scans
// net.Interfaces for a Fortinet tunnel with an IPv4 address and the UP flag.
// Used in Status() polling — must be fast.
func probeFortiInterface() (string, net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil
	}
	// On Unix, tunnel names are distinctive (fortissl, vpnssl0, etc.).
	// On Windows, names are generic but we check all UP interfaces for a VPN-like
	// IPv4 address. If we previously knew the interface name, fortiIfaceUp is used
	// instead. This path is for discovering a NEW tunnel we haven't seen yet.
	unixCandidates := []string{"fortissl", "vpnssl0", "ssl0", "ppp0"}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		for _, c := range unixCandidates {
			if strings.HasPrefix(name, c) {
				addrs, _ := iface.Addrs()
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
						return iface.Name, ipnet.IP
					}
				}
			}
		}
	}
	return "", nil
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

	// Read byte counters from the OS interface.
	if a.tunnelIface != "" {
		info.BytesSent, info.BytesReceived = ifaceByteCounters(a.tunnelIface)
	}

	// Read DNS servers assigned to the tunnel interface by FortiClient.
	if a.tunnelIface != "" {
		info.DNS = ifaceDNSServers(a.tunnelIface)
	}

	return info, nil
}
