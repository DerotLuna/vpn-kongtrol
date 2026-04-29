package wireguard

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// interfaceFromConfig derives the WireGuard interface name from a config file path.
// /etc/wireguard/wg0.conf  →  wg0
// ~/.kongtrol/configs/work.conf  →  work
func interfaceFromConfig(configPath string) string {
	base := filepath.Base(configPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// parseConfigAddress reads the [Interface] Address line from a .conf file.
// Returns the host IP (without prefix length).
func parseConfigAddress(configPath string) (net.IP, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inInterface := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(line, "[Interface]") {
			inInterface = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inInterface = false
		}
		if inInterface && strings.HasPrefix(strings.ToLower(line), "address") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}
			// May be "10.0.0.2/24" — take the host part.
			addr := strings.TrimSpace(strings.Split(parts[1], ",")[0])
			ip, _, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
			}
			return ip, nil
		}
	}
	return nil, fmt.Errorf("wireguard: no Address found in %s", configPath)
}

// parseConfigDNS reads the [Interface] DNS line from a .conf file.
func parseConfigDNS(configPath string) []net.IP {
	f, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inInterface := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(line, "[Interface]") {
			inInterface = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inInterface = false
		}
		if inInterface && strings.HasPrefix(strings.ToLower(line), "dns") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}
			var ips []net.IP
			for _, s := range strings.Split(parts[1], ",") {
				s = strings.TrimSpace(s)
				if ip := net.ParseIP(s); ip != nil {
					ips = append(ips, ip)
				}
			}
			return ips
		}
	}
	return nil
}

// ParsePeerPublicKey reads the first [Peer] PublicKey from a WireGuard .conf file.
// Exported because the PolicyResolver (internal/monitor) calls it.
func ParsePeerPublicKey(configPath string) (string, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inPeer := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(line, "[Peer]") {
			inPeer = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPeer = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(line), "publickey") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSpace(parts[1])
			if key != "" {
				return key, nil
			}
		}
	}
	return "", fmt.Errorf("wireguard: no [Peer] PublicKey found in %s", configPath)
}

// ParseEndpoint reads the first [Peer] Endpoint IP from a WireGuard .conf file.
// Returns only the IP portion (no port). Exported for PolicyResolver.
func ParseEndpoint(configPath string) (net.IP, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inPeer := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(line, "[Peer]") {
			inPeer = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPeer = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(line), "endpoint") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}
			hostport := strings.TrimSpace(parts[1])
			// Handle [IPv6]:port and IPv4:port
			host, _, err := net.SplitHostPort(hostport)
			if err != nil {
				// Maybe bare IP without port
				host = hostport
			}
			ip := net.ParseIP(host)
			if ip != nil {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("wireguard: no [Peer] Endpoint found in %s", configPath)
}

// ParseConfigAddress is the exported form of parseConfigAddress for cross-package use.
func ParseConfigAddress(configPath string) (net.IP, error) { return parseConfigAddress(configPath) }

// ParseConfigDNS is the exported form of parseConfigDNS for cross-package use.
func ParseConfigDNS(configPath string) []net.IP { return parseConfigDNS(configPath) }

// parseHandshake checks if wg show output contains a recent handshake (< 3 min).
func parseHandshake(wgShowOutput string) bool {
	for _, line := range strings.Split(wgShowOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "latest handshake:") {
			// "latest handshake: 1 minute, 30 seconds ago"
			// "latest handshake: 2 minutes, 5 seconds ago"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				return false
			}
			// Very rough heuristic: handshake is recent if < 3 minutes mentioned.
			text := strings.ToLower(parts[1])
			if strings.Contains(text, "second") {
				return true
			}
			if strings.Contains(text, "minute") {
				// Extract the minute number.
				words := strings.Fields(text)
				for i, w := range words {
					if w == "minute" || w == "minutes" {
						if i > 0 {
							n, err := strconv.Atoi(words[i-1])
							if err == nil && n < 3 {
								return true
							}
						}
					}
				}
			}
			return false
		}
	}
	return false
}

// parseTransfer extracts (sent, received) byte counts from `wg show <iface> transfer`.
func parseTransfer(raw string) (sent, recv uint64) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 {
		return
	}
	// Take the first peer's line: "<pubkey>\t<rx>\t<tx>"
	fields := strings.Fields(lines[0])
	if len(fields) >= 3 {
		recv, _ = strconv.ParseUint(fields[1], 10, 64)
		sent, _ = strconv.ParseUint(fields[2], 10, 64)
	}
	return
}

// ifaceExists returns true if a network interface with the given name exists.
func ifaceExists(name string) bool {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Name == name {
			return true
		}
	}
	return false
}

// waitForInterface polls until the WireGuard interface appears in net.Interfaces().
// On timeout it includes the names of all current interfaces to aid diagnosis.
func waitForInterface(ifaceName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			if iface.Name == ifaceName {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Build a list of current interface names to help diagnose naming mismatches.
	ifaces, _ := net.Interfaces()
	names := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		names = append(names, iface.Name)
	}
	return fmt.Errorf("wireguard: interface %q did not appear within %s (present interfaces: %v)",
		ifaceName, timeout, names)
}
