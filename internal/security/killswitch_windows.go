//go:build windows

package security

import (
	"fmt"
	"os/exec"
	"sync"
)

const ksRuleBlock = "KongtrolKillSwitchBlock"
const ksRuleAllow = "KongtrolKillSwitchAllow"
const ksRuleLAN = "KongtrolKillSwitchLAN"

type windowsKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the Windows kill switch implementation.
// Uses Windows Firewall (netsh advfirewall) for broad compatibility.
// For WFP-level control (survives reboot), upgrade to iphlpapi/WFP calls.
func NewKillSwitch() KillSwitch {
	return &windowsKillSwitch{}
}

func (k *windowsKillSwitch) Enable(tunnelInterface string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Remove any stale rules from a previous session.
	_ = k.removeRules()

	// Block all outbound traffic.
	if err := netshFW("add", "rule",
		"name="+ksRuleBlock,
		"dir=out", "action=block",
		"enable=yes", "profile=any",
	); err != nil {
		return fmt.Errorf("kill switch: add block rule: %w", err)
	}

	// Allow traffic on the tunnel interface.
	if tunnelInterface != "" {
		if err := netshFW("add", "rule",
			"name="+ksRuleAllow,
			"dir=out", "action=allow",
			"enable=yes", "profile=any",
			"interface="+tunnelInterface,
		); err != nil {
			return fmt.Errorf("kill switch: add tunnel allow rule: %w", err)
		}
	}

	// Allow LAN traffic if requested.
	if allowLAN {
		for _, subnet := range []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"} {
			_ = netshFW("add", "rule",
				"name="+ksRuleLAN,
				"dir=out", "action=allow",
				"enable=yes", "profile=any",
				"remoteip="+subnet,
			)
		}
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
	for _, name := range []string{ksRuleBlock, ksRuleAllow, ksRuleLAN} {
		// Ignore errors — rules may not exist.
		_ = netshFW("delete", "rule", "name="+name)
	}
	return nil
}

func netshFW(args ...string) error {
	fullArgs := append([]string{"advfirewall", "firewall"}, args...)
	cmd := exec.Command("netsh", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh advfirewall %v: %w (%s)", args, err, out)
	}
	return nil
}
