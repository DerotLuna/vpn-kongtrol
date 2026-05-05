//go:build windows

package security

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

const ksRuleBlock = "KongtrolKillSwitchBlock"
const ksRuleAllow = "KongtrolKillSwitchAllow"
const ksRuleEndpoint = "KongtrolKillSwitchEndpoint"
const ksRuleLAN = "KongtrolKillSwitchLAN"
const ksRuleLoopback = "KongtrolKillSwitchLoopback"

type windowsKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the Windows kill switch implementation.
// Uses Windows Firewall (netsh advfirewall) for broad compatibility.
func NewKillSwitch() KillSwitch {
	return &windowsKillSwitch{}
}

// Enable activates the kill switch. tunnelSpec format:
//
//	"iface1,iface2|endpoint1,endpoint2"
//
// Left of | = tunnel interface names (their local IPs are allowed as source).
// Right of | = VPN endpoint IPs (allowed as destination so encrypted tunnel works).
// Either side may be empty.
//
// If not running elevated, triggers a single UAC prompt to add all firewall rules.
func (k *windowsKillSwitch) Enable(tunnelSpec string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Remove any stale rules from a previous session.
	_ = k.removeRules()

	// Parse tunnelSpec.
	var ifaceNames, endpointIPs []string
	parts := strings.SplitN(tunnelSpec, "|", 2)
	if parts[0] != "" {
		ifaceNames = strings.Split(parts[0], ",")
	}
	if len(parts) > 1 && parts[1] != "" {
		endpointIPs = strings.Split(parts[1], ",")
	}

	// Collect all netsh commands to batch into a single elevated execution.
	var cmds [][]string

	// Block all outbound traffic.
	cmds = append(cmds, []string{"advfirewall", "firewall", "add", "rule",
		"name=" + ksRuleBlock,
		"dir=out", "action=block",
		"enable=yes", "profile=any"})

	// Allow loopback (127.0.0.0/8).
	cmds = append(cmds, []string{"advfirewall", "firewall", "add", "rule",
		"name=" + ksRuleLoopback,
		"dir=out", "action=allow",
		"enable=yes", "profile=any",
		"remoteip=127.0.0.0/8"})

	// Allow traffic FROM VPN-assigned IPs (tunnel interfaces).
	var localIPs []string
	for _, name := range ifaceNames {
		localIPs = append(localIPs, getInterfaceIPs(name)...)
	}
	if len(localIPs) > 0 {
		cmds = append(cmds, []string{"advfirewall", "firewall", "add", "rule",
			"name=" + ksRuleAllow,
			"dir=out", "action=allow",
			"enable=yes", "profile=any",
			"localip=" + strings.Join(localIPs, ",")})
	}

	// Allow traffic TO VPN endpoint IPs (so encrypted tunnel can reach the server).
	if len(endpointIPs) > 0 {
		cmds = append(cmds, []string{"advfirewall", "firewall", "add", "rule",
			"name=" + ksRuleEndpoint,
			"dir=out", "action=allow",
			"enable=yes", "profile=any",
			"remoteip=" + strings.Join(endpointIPs, ",")})
	}

	// Allow LAN traffic if requested.
	if allowLAN {
		for _, subnet := range []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"} {
			cmds = append(cmds, []string{"advfirewall", "firewall", "add", "rule",
				"name=" + ksRuleLAN,
				"dir=out", "action=allow",
				"enable=yes", "profile=any",
				"remoteip=" + subnet})
		}
	}

	if err := runNetshElevated(cmds); err != nil {
		return fmt.Errorf("kill switch: %w", err)
	}

	k.enabled = true
	return nil
}

func (k *windowsKillSwitch) Disable() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	err := k.removeRules()
	k.enabled = false
	return err
}

func (k *windowsKillSwitch) IsEnabled() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.enabled
}

func (k *windowsKillSwitch) removeRules() error {
	var cmds [][]string
	for _, name := range []string{ksRuleBlock, ksRuleAllow, ksRuleEndpoint, ksRuleLAN, ksRuleLoopback} {
		cmds = append(cmds, []string{"advfirewall", "firewall", "delete", "rule", "name=" + name})
	}
	// Best-effort cleanup — ignore errors (rules may not exist).
	_ = runNetshElevated(cmds)
	return nil
}

// getInterfaceIPs returns the IPv4 addresses assigned to the named interface.
func getInterfaceIPs(name string) []string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	var ips []string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}
	return ips
}
