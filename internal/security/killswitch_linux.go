//go:build linux

package security

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// iptablesTimeout bounds every iptables invocation. Kill-switch enable/
// disable runs on the daemon's shutdown/cleanup path (including SIGTERM); a
// hung iptables (lock contention, unusual system state) must not be able to
// block process exit or leave OUTPUT rules in an indeterminate state
// indefinitely.
const iptablesTimeout = 5 * time.Second

type linuxKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the Linux kill switch using iptables.
func NewKillSwitch() KillSwitch {
	return &linuxKillSwitch{}
}

func (k *linuxKillSwitch) Enable(tunnelSpec string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Parse tunnelSpec: "iface1,iface2|endpoint1,endpoint2"
	var ifaceNames, endpointIPs []string
	parts := strings.SplitN(tunnelSpec, "|", 2)
	if parts[0] != "" {
		ifaceNames = strings.Split(parts[0], ",")
	}
	if len(parts) > 1 && parts[1] != "" {
		endpointIPs = strings.Split(parts[1], ",")
	}

	_ = k.removeRules() // clean up stale rules

	// Allow established connections.
	if err := ipt("-A", "OUTPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("kill switch: allow established: %w", err)
	}
	// Allow loopback.
	if err := ipt("-A", "OUTPUT", "-o", "lo", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("kill switch: allow loopback: %w", err)
	}
	// Allow tunnel interfaces.
	for _, iface := range ifaceNames {
		if err := ipt("-A", "OUTPUT", "-o", iface, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("kill switch: allow tunnel %s: %w", iface, err)
		}
	}
	// Allow VPN endpoint IPs (so encrypted tunnel can reach the server).
	for _, ep := range endpointIPs {
		_ = ipt("-A", "OUTPUT", "-d", ep, "-j", "ACCEPT")
	}
	// Allow LAN ranges.
	if allowLAN {
		for _, subnet := range []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"} {
			_ = ipt("-A", "OUTPUT", "-d", subnet, "-j", "ACCEPT")
		}
	}
	// Block everything else.
	if err := ipt("-A", "OUTPUT", "-j", "DROP"); err != nil {
		return fmt.Errorf("kill switch: add drop rule: %w", err)
	}

	k.enabled = true
	return nil
}

func (k *linuxKillSwitch) Disable() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	err := k.removeRules()
	k.enabled = false
	return err
}

func (k *linuxKillSwitch) IsEnabled() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.enabled
}

// removeRules flushes the OUTPUT chain rules added by kongtrol.
// Uses iptables -F OUTPUT which removes ALL output rules — fine for a VPN tool
// where we own the OUTPUT chain. If coexisting with other firewall rules,
// switch to a named chain with -N kongtrol and -X kongtrol on cleanup.
func (k *linuxKillSwitch) removeRules() error {
	return ipt("-F", "OUTPUT")
}

func ipt(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), iptablesTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "iptables", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("iptables %v: timed out after %s (%s)", args, iptablesTimeout, out)
		}
		return fmt.Errorf("iptables %v: %w (%s)", args, err, out)
	}
	return nil
}
