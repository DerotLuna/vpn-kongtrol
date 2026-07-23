package protonvpn

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// connect calls protonvpn-cli to connect to a specific server.
// Requires protonvpn-cli v3.x installed and authenticated.
func connect(server, protocol string) error {
	args := []string{"connect", "--cc", server}
	if protocol != "" {
		args = append(args, "-p", protocol)
	}
	return cli(args...)
}

// disconnect calls protonvpn-cli to disconnect.
func disconnect() error {
	return cli("disconnect")
}

// statusOutput returns the raw output of `protonvpn-cli status`.
// We detect the version and use --json if available (v3.10+).
func statusOutput() (string, error) {
	// Try JSON mode first (v3.10+).
	out, err := cliOutput("status", "--json")
	if err == nil && strings.HasPrefix(strings.TrimSpace(out), "{") {
		return out, nil
	}
	// Fallback to human-readable.
	return cliOutput("status")
}

// parseStatus extracts the connection state from protonvpn-cli status output.
// Handles both JSON (v3.10+) and human-readable (older) formats.
func parseStatus(raw string) (connected bool, serverIP, assignedIP string) {
	raw = strings.TrimSpace(raw)

	// JSON format: {"Status":"Connected","ServerIP":"...","IP":"..."}
	if strings.HasPrefix(raw, "{") {
		connected = strings.Contains(raw, `"Status":"Connected"`)
		serverIP = extractJSON(raw, "ServerIP")
		assignedIP = extractJSON(raw, "IP")
		return
	}

	// Human-readable: look for "Status:" and "IP:" lines.
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			connected = strings.Contains(strings.ToLower(line), "connected")
		}
		if strings.HasPrefix(line, "Server IP:") {
			serverIP = strings.TrimSpace(strings.TrimPrefix(line, "Server IP:"))
		}
		if strings.HasPrefix(line, "IP:") {
			assignedIP = strings.TrimSpace(strings.TrimPrefix(line, "IP:"))
		}
	}
	return
}

func extractJSON(s, key string) string {
	needle := `"` + key + `":"`
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func cli(args ...string) error {
	cmd := exec.Command("protonvpn-cli", args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protonvpn-cli %v: %w\n%s", args, err, out.String())
	}
	return nil
}

func cliOutput(args ...string) (string, error) {
	cmd := exec.Command("protonvpn-cli", args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
