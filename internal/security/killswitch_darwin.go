//go:build darwin

package security

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const anchorName = "kongtrol"
const anchorDir = "/etc/pf.anchors"

type darwinKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the macOS kill switch using pf (packet filter).
func NewKillSwitch() KillSwitch {
	return &darwinKillSwitch{}
}

func (k *darwinKillSwitch) Enable(tunnelInterface string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	rules := "block out all\n"
	rules += "pass out on lo0\n"

	if tunnelInterface != "" {
		rules += fmt.Sprintf("pass out on %s\n", tunnelInterface)
	}
	if allowLAN {
		rules += "pass out to 192.168.0.0/16\n"
		rules += "pass out to 10.0.0.0/8\n"
		rules += "pass out to 172.16.0.0/12\n"
	}

	anchorFile := filepath.Join(anchorDir, anchorName)
	if err := os.WriteFile(anchorFile, []byte(rules), 0644); err != nil {
		return fmt.Errorf("kill switch: write pf anchor: %w", err)
	}

	if err := pfctl("-a", anchorName, "-f", anchorFile); err != nil {
		return fmt.Errorf("kill switch: load pf anchor: %w", err)
	}

	k.enabled = true
	return nil
}

func (k *darwinKillSwitch) Disable() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Flush the kongtrol anchor.
	_ = pfctl("-a", anchorName, "-F", "all")
	k.enabled = false
	return nil
}

func (k *darwinKillSwitch) IsEnabled() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.enabled
}

func pfctl(args ...string) error {
	cmd := exec.Command("pfctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pfctl %v: %w (%s)", args, err, out)
	}
	return nil
}
