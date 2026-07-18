//go:build windows

package security

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// netshQueryTimeout bounds the read-only netsh query used to snapshot the
// current DNS servers before overwriting them.
const netshQueryTimeout = 5 * time.Second

type windowsDNSGuard struct {
	mu      sync.Mutex
	active  bool
	iface   string
	prevDNS []string // stored as strings for easy restore
}

// NewDNSGuard returns the Windows DNS guard implementation.
// Uses netsh to set/restore DNS resolver per interface.
// If not running elevated, triggers a UAC prompt.
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

	// Batch all netsh commands into a single elevated execution.
	var cmds [][]string

	// Set primary DNS.
	cmds = append(cmds, []string{"interface", "ip", "set", "dns", iface, dnsServers[0].String(), "static"})

	// Add secondary DNS servers.
	for _, srv := range dnsServers[1:] {
		cmds = append(cmds, []string{"interface", "ip", "add", "dns", iface, srv.String()})
	}

	if err := runNetshElevated(cmds); err != nil {
		return fmt.Errorf("dnsguard: set DNS: %w", err)
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

	var cmds [][]string

	if len(g.prevDNS) == 0 {
		// Restore to DHCP-assigned DNS.
		cmds = append(cmds, []string{"interface", "ip", "set", "dns", g.iface, "dhcp"})
	} else {
		cmds = append(cmds, []string{"interface", "ip", "set", "dns", g.iface, g.prevDNS[0], "static"})
		for _, srv := range g.prevDNS[1:] {
			cmds = append(cmds, []string{"interface", "ip", "add", "dns", g.iface, srv})
		}
	}

	if err := runNetshElevated(cmds); err != nil {
		g.active = false // mark inactive even on error to avoid retry loops
		return fmt.Errorf("dnsguard: restore DNS: %w", err)
	}

	g.active = false
	return nil
}

func (g *windowsDNSGuard) IsActive() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active
}

// currentDNSWindows reads the current DNS servers for an interface via netsh.
// This is a read-only query that doesn't require elevation.
func currentDNSWindows(iface string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshQueryTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "netsh", "interface", "ip", "show", "dns", iface)
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
