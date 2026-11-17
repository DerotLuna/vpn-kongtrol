//go:build darwin

package appattach

import (
	"os/exec"
)

// osServiceTunnel finds a connected macOS network service (Network
// Extension VPN of a provider app) whose name matches a pattern and
// returns its utun interface when `scutil --nc status` reports it. When
// the interface cannot be read the service name stands in for it so the
// tunnel is still attached (status / watchdog work; DNS guard may not).
func osServiceTunnel(patterns []string) (Tunnel, bool) {
	if len(patterns) == 0 {
		return Tunnel{}, false
	}
	out, err := exec.Command("scutil", "--nc", "list").Output()
	if err != nil {
		return Tunnel{}, false
	}
	name, ok := pickService(parseNCList(string(out)), patterns)
	if !ok {
		return Tunnel{}, false
	}
	t := Tunnel{}
	if st, err := exec.Command("scutil", "--nc", "status", name).Output(); err == nil {
		t = parseNCStatus(string(st))
	}
	if t.Name == "" {
		t.Name = name
	}
	return t, true
}
