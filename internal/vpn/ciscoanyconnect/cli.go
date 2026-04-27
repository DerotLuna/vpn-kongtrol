package ciscoanyconnect

import (
	"fmt"
	"net"
	"runtime"
	"strings"
)

// binaryPath returns the platform-specific path to the AnyConnect CLI binary.
func binaryPath() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Program Files\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`
	case "darwin":
		return "/opt/cisco/anyconnect/bin/vpn"
	default: // linux
		return "/opt/cisco/anyconnect/bin/vpn"
	}
}

// connect initiates a Cisco AnyConnect connection by piping credentials to the CLI.
// AnyConnect's interactive session expects: username\npassword\ny\n (accept banner).
func connect(host, username, password string) error {
	input := fmt.Sprintf("%s\n%s\ny\n", username, password)
	out, err := runCmdWithStdin(binaryPath(), []string{"connect", host}, input)
	if err != nil {
		return fmt.Errorf("vpncli connect: %w\n%s", err, out)
	}
	if strings.Contains(strings.ToLower(out), "error") {
		return fmt.Errorf("vpncli connect failed: %s", out)
	}
	return nil
}

// disconnect tears down the active AnyConnect tunnel.
func disconnect() error {
	out, err := runCmd(binaryPath(), "disconnect")
	if err != nil {
		return fmt.Errorf("vpncli disconnect: %w\n%s", err, out)
	}
	return nil
}

// statusOutput returns the raw output of `vpncli status`.
func statusOutput() (string, error) {
	return runCmd(binaryPath(), "status")
}

// statsOutput returns the raw output of `vpncli stats`.
func statsOutput() (string, error) {
	return runCmd(binaryPath(), "stats")
}

// parseStatus extracts connection state from vpncli status output.
// Returns (connected, clientIP, serverIP).
func parseStatus(raw string) (connected bool, clientIP string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "state:") && strings.Contains(lower, "connected") {
			connected = true
		}
		// "Client Address: 10.x.x.x"
		if strings.HasPrefix(lower, "client address:") {
			clientIP = strings.TrimSpace(strings.TrimPrefix(line, strings.Split(line, ":")[0]+":"))
		}
	}
	return
}

// parseDNS extracts DNS server IPs from `vpncli stats` output.
// Line format: "DNS Servers: 10.0.1.53, 10.0.1.54"
func parseDNS(statsRaw string) []net.IP {
	for _, line := range strings.Split(statsRaw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "dns servers:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			var dns []net.IP
			for _, s := range strings.Split(parts[1], ",") {
				s = strings.TrimSpace(s)
				if ip := net.ParseIP(s); ip != nil {
					dns = append(dns, ip)
				}
			}
			return dns
		}
	}
	return nil
}

// detectVersion returns the AnyConnect version string.
func detectVersion() string {
	out, err := runCmd(binaryPath(), "--version")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "version") || strings.Contains(line, "AnyConnect") {
				return strings.TrimSpace(line)
			}
		}
	}
	return "unknown"
}
