//go:build darwin

package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// pfctlTimeout bounds every pfctl invocation. Kill-switch enable/disable
// runs on the daemon's shutdown/cleanup path (including SIGTERM); a hung
// pfctl (lock contention, unusual system state) must not be able to block
// process exit or leave the anchor in an indeterminate state indefinitely.
const pfctlTimeout = 5 * time.Second

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

func (k *darwinKillSwitch) Enable(tunnelSpec string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Parse tunnelSpec: "iface1,iface2|endpoint1,endpoint2"
	var ifaceNames, endpointIPs []string
	specParts := strings.SplitN(tunnelSpec, "|", 2)
	if specParts[0] != "" {
		ifaceNames = strings.Split(specParts[0], ",")
	}
	if len(specParts) > 1 && specParts[1] != "" {
		endpointIPs = strings.Split(specParts[1], ",")
	}

	rules := "block out all\n"
	rules += "pass out on lo0\n"

	for _, iface := range ifaceNames {
		rules += fmt.Sprintf("pass out on %s\n", iface)
	}
	for _, ep := range endpointIPs {
		rules += fmt.Sprintf("pass out to %s\n", ep)
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
	ctx, cancel := context.WithTimeout(context.Background(), pfctlTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pfctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("pfctl %v: timed out after %s (%s)", args, pfctlTimeout, out)
		}
		return fmt.Errorf("pfctl %v: %w (%s)", args, err, out)
	}
	return nil
}
