//go:build linux || darwin

package security

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

const resolvConf = "/etc/resolv.conf"
const resolvBackup = "/etc/resolv.conf.kongtrol.bak"

type unixDNSGuard struct {
	mu     sync.Mutex
	active bool
	iface  string
}

// NewDNSGuard returns the Unix DNS guard (Linux/macOS).
// Linux: rewrites /etc/resolv.conf (with backup).
// macOS: uses networksetup -setdnsservers per service.
func NewDNSGuard() DNSGuard {
	return &unixDNSGuard{}
}

func (g *unixDNSGuard) Apply(iface string, dnsServers []net.IP) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(dnsServers) == 0 {
		return fmt.Errorf("dnsguard: no DNS servers provided")
	}
	g.iface = iface

	switch runtime.GOOS {
	case "darwin":
		return g.applyDarwin(iface, dnsServers)
	default:
		return g.applyLinux(dnsServers)
	}
}

func (g *unixDNSGuard) Restore() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.active {
		return nil
	}

	var err error
	switch runtime.GOOS {
	case "darwin":
		err = g.restoreDarwin(g.iface)
	default:
		err = g.restoreLinux()
	}
	if err == nil {
		g.active = false
	}
	return err
}

func (g *unixDNSGuard) IsActive() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active
}

// ── Linux: /etc/resolv.conf ──────────────────────────────────────────────────

func (g *unixDNSGuard) applyLinux(dnsServers []net.IP) error {
	// Backup current resolv.conf.
	current, err := os.ReadFile(resolvConf)
	if err != nil {
		return fmt.Errorf("dnsguard: read resolv.conf: %w", err)
	}
	if err := os.WriteFile(resolvBackup, current, 0644); err != nil {
		return fmt.Errorf("dnsguard: backup resolv.conf: %w", err)
	}

	// Write new resolv.conf.
	var sb strings.Builder
	sb.WriteString("# Managed by vpn-kongtrol — original backed up at " + resolvBackup + "\n")
	for _, srv := range dnsServers {
		sb.WriteString("nameserver " + srv.String() + "\n")
	}

	if err := os.WriteFile(resolvConf, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("dnsguard: write resolv.conf: %w", err)
	}

	g.active = true
	return nil
}

func (g *unixDNSGuard) restoreLinux() error {
	backup, err := os.ReadFile(resolvBackup)
	if err != nil {
		return fmt.Errorf("dnsguard: read backup: %w", err)
	}
	if err := os.WriteFile(resolvConf, backup, 0644); err != nil {
		return fmt.Errorf("dnsguard: restore resolv.conf: %w", err)
	}
	_ = os.Remove(resolvBackup)
	return nil
}

// ── macOS: networksetup ───────────────────────────────────────────────────────

func (g *unixDNSGuard) applyDarwin(iface string, dnsServers []net.IP) error {
	// Resolve iface name to a network service name (e.g. "Wi-Fi", "Ethernet").
	service, err := darwinServiceForInterface(iface)
	if err != nil {
		// Fall back to writing resolv.conf (works on macOS too).
		return g.applyLinux(dnsServers)
	}

	args := []string{"-setdnsservers", service}
	for _, srv := range dnsServers {
		args = append(args, srv.String())
	}

	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnsguard: networksetup %v: %w (%s)", args, err, out)
	}

	g.active = true
	return nil
}

func (g *unixDNSGuard) restoreDarwin(iface string) error {
	service, err := darwinServiceForInterface(iface)
	if err != nil {
		return g.restoreLinux()
	}

	// Restore to DHCP-assigned DNS.
	cmd := exec.Command("networksetup", "-setdnsservers", service, "empty")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnsguard: restore DNS: %w (%s)", err, out)
	}
	return nil
}

// darwinServiceForInterface maps a BSD interface name (e.g. "utun0") to
// a macOS network service name (e.g. "VPN (L2TP)") using networksetup.
func darwinServiceForInterface(iface string) (string, error) {
	cmd := exec.Command("networksetup", "-listallhardwareports")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse the output looking for:
	// Hardware Port: <service>
	// Device: <iface>
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Device: "+iface) && i > 0 {
			for j := i - 1; j >= 0; j-- {
				if strings.HasPrefix(lines[j], "Hardware Port:") {
					return strings.TrimSpace(strings.TrimPrefix(lines[j], "Hardware Port:")), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no service found for interface %q", iface)
}
