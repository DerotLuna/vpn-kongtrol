package tailscale

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// tsStatusJSON is a minimal subset of `tailscale status --json`.
type tsStatusJSON struct {
	BackendState string   `json:"BackendState"` // "Running", "Stopped", "NeedsLogin"
	TailscaleIPs []string `json:"TailscaleIPs"` // node's Tailscale IPs
	Self         struct {
		DNSName      string   `json:"DNSName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
	} `json:"Self"`
}

func tsUp(authKey, exitNode string) error {
	binary := binaryPath()
	args := []string{"up"}
	if authKey != "" {
		args = append(args, "--authkey", authKey)
	}
	if exitNode != "" {
		args = append(args, "--exit-node", exitNode)
	}
	out, err := runCmd(binary, args...)
	if err != nil {
		// "already running" is not an error.
		if strings.Contains(strings.ToLower(out+err.Error()), "already") {
			return nil
		}
		return fmt.Errorf("tailscale up: %w\n%s", err, out)
	}
	return nil
}

// tsDown pauses the Tailscale connection (does NOT log out / de-authenticate).
func tsDown() error {
	out, err := runCmd(binaryPath(), "down")
	if err != nil {
		return fmt.Errorf("tailscale down: %w\n%s", err, out)
	}
	return nil
}

// tsStatus returns the parsed status JSON.
func tsStatus() (*tsStatusJSON, error) {
	raw, err := runCmd(binaryPath(), "status", "--json")
	if err != nil && raw == "" {
		return nil, fmt.Errorf("tailscale status: daemon may not be running: %w", err)
	}
	var s tsStatusJSON
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil, fmt.Errorf("tailscale status: parse JSON: %w", err)
	}
	return &s, nil
}

// parseAssignedIP returns the first Tailscale IPv4 from the status.
func parseAssignedIP(s *tsStatusJSON) net.IP {
	ips := s.Self.TailscaleIPs
	if len(ips) == 0 {
		ips = s.TailscaleIPs
	}
	for _, ipStr := range ips {
		if ip := net.ParseIP(ipStr); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				return v4
			}
		}
	}
	return nil
}

func detectVersion() string {
	out, err := runCmd(binaryPath(), "version")
	if err == nil {
		return strings.TrimSpace(out)
	}
	return "unknown"
}
