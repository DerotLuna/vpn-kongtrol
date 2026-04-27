//go:build linux

package security

import (
	"fmt"
	"os/exec"
	"sync"
)

type linuxKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the Linux kill switch using iptables.
func NewKillSwitch() KillSwitch {
	return &linuxKillSwitch{}
}

func (k *linuxKillSwitch) Enable(tunnelInterface string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	_ = k.removeRules() // clean up stale rules

	// Allow established connections.
	if err := ipt("-A", "OUTPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("kill switch: allow established: %w", err)
	}
	// Allow loopback.
	if err := ipt("-A", "OUTPUT", "-o", "lo", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("kill switch: allow loopback: %w", err)
	}
	// Allow tunnel interface.
	if tunnelInterface != "" {
		if err := ipt("-A", "OUTPUT", "-o", tunnelInterface, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("kill switch: allow tunnel %s: %w", tunnelInterface, err)
		}
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
	cmd := exec.Command("iptables", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables %v: %w (%s)", args, err, out)
	}
	return nil
}
