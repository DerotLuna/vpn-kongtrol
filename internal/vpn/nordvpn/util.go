package nordvpn

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// matcher finds the NordVPN tunnel by friendly name or adapter description:
// NordLynx ("NordLynx", Linux "nordlynx"), the OpenVPN TAP adapter
// ("TAP-NordVPN Windows Adapter V9", named "Ethernet N" on some systems)
// and the Linux OpenVPN device "nordtun". On macOS the app's tunnel is a
// generic utunN, so it is found by the app's network service
// ("NordVPN ...") in `scutil --nc list` (best effort). `interface:`
// overrides all of this.
var matcher = appattach.Matcher{
	Patterns: []string{"*nordlynx*", "*nordvpn*", "nordtun*"},
	Services: []string{"nordvpn*"},
}

// runCmd executes the NordVPN client and returns its combined output.
func runCmd(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// findBinary resolves the client binary: the profile's binary_path override
// first, then the platform's standard locations (absolute paths or PATH
// names).
func findBinary(override string, locations []string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", fmt.Errorf("binary_path %s: %w", override, err)
		}
		return override, nil
	}
	for _, loc := range locations {
		if fi, err := os.Stat(loc); err == nil && fi.Mode().IsRegular() {
			return loc, nil
		}
		if p, err := exec.LookPath(loc); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("NordVPN client not found (looked in %v) — install NordVPN first", locations)
}
