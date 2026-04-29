//go:build windows

package security

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type windowsDNSGuard struct {
	mu         sync.Mutex
	active     bool
	iface      string
	prevDNS    []string // stored as strings for easy restore
}

// NewDNSGuard returns the Windows DNS guard implementation.
// Uses netsh to set/restore DNS resolver per interface.
func NewDNSGuard() DNSGuard {
	return &windowsDNSGuard{}
}

func (g *windowsDNSGuard) Apply(iface string, dnsServers []net.IP) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(dnsServers) == 0 {
		return fmt.Errorf("dnsguard: no DNS servers provided")
	}

	// Save current DNS for restore.
	current, _ := currentDNSWindows(iface)
	g.prevDNS = current
	g.iface = iface

	// Set primary DNS.
	if err := netshDNS(iface, "set", dnsServers[0].String(), "static"); err != nil {
		return fmt.Errorf("dnsguard: set primary DNS: %w", err)
	}

	// Add secondary DNS servers.
	for _, srv := range dnsServers[1:] {
		_ = netshDNS(iface, "add", srv.String(), "")
	}

	g.active = true
	return nil
}

func (g *windowsDNSGuard) Restore() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.active {
		return nil
	}

	if len(g.prevDNS) == 0 {
		// Restore to DHCP-assigned DNS.
		if err := netshDNS(g.iface, "set", "dhcp", ""); err != nil {
			return fmt.Errorf("dnsguard: restore DHCP DNS: %w", err)
		}
	} else {
		if err := netshDNS(g.iface, "set", g.prevDNS[0], "static"); err != nil {
			return fmt.Errorf("dnsguard: restore primary DNS: %w", err)
		}
		for _, srv := range g.prevDNS[1:] {
			_ = netshDNS(g.iface, "add", srv, "")
		}
	}

	g.active = false
	return nil
}

func (g *windowsDNSGuard) IsActive() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active
}

func netshDNS(iface, action, addr, mode string) error {
	args := []string{"interface", "ip", action, "dns", iface, addr}
	if mode != "" {
		args = append(args, mode)
	}
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %v: %w (%s)", args, err, out)
	}
	return nil
}

// currentDNSWindows reads the current DNS servers for an interface via netsh.
func currentDNSWindows(iface string) ([]string, error) {
	cmd := exec.Command("netsh", "interface", "ip", "show", "dns", iface)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if ip := net.ParseIP(fields[len(fields)-1]); ip != nil {
			servers = append(servers, ip.String())
		}
	}
	return servers, nil
}
