package cloudflarewarp

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

// Hardcoded Cloudflare WARP DNS servers (1.1.1.1 / 1.0.0.1).
var warpDNS = []net.IP{
	net.ParseIP("1.1.1.1"),
	net.ParseIP("1.0.0.1"),
}

func warpConnect() error {
	binary, err := warpBinary()
	if err != nil {
		return err
	}
	out, err := runCmd(binary, "connect")
	if err != nil {
		if strings.Contains(strings.ToLower(out), "registration missing") ||
			strings.Contains(strings.ToLower(out), "not registered") {
			return fmt.Errorf("cloudflarewarp: not registered — run 'warp-cli register' first")
		}
		return fmt.Errorf("warp-cli connect: %w\n%s", err, out)
	}
	return nil
}

func warpDisconnect() error {
	binary, err := warpBinary()
	if err != nil {
		return err
	}
	out, err := runCmd(binary, "disconnect")
	if err != nil {
		return fmt.Errorf("warp-cli disconnect: %w\n%s", err, out)
	}
	return nil
}

func warpStatusOutput() (string, error) {
	binary, err := warpBinary()
	if err != nil {
		return "", err
	}
	return runCmd(binary, "status")
}

// parseWarpStatus extracts the connection state from `warp-cli status` output.
// Handles multiple "Status update:" lines by taking the last one.
func parseWarpStatus(raw string) string {
	last := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "status update:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				last = strings.TrimSpace(parts[1])
			}
		}
	}
	return strings.ToLower(last)
}

// warpInterface finds the active WARP tunnel interface.
// Interface names: "CloudflareWARP" (Windows), "warp0" (Linux), "utun<n>" (macOS).
func warpInterface() (string, net.IP) {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "warp") || strings.Contains(name, "cloudflare") {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					return iface.Name, ipnet.IP
				}
			}
		}
	}
	// macOS: WARP uses a utun interface; scan for a new utun after connect.
	if runtime.GOOS == "darwin" {
		return scanUtun()
	}
	return "", nil
}

// scanUtun finds a recently added utun interface (heuristic: has an IP, is not loopback).
func scanUtun() (string, net.IP) {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if !strings.HasPrefix(iface.Name, "utun") {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return iface.Name, ipnet.IP
			}
		}
	}
	return "", nil
}

// waitConnected polls warp-cli status until "connected" appears or timeout.
func waitConnected(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		raw, err := warpStatusOutput()
		if err == nil {
			state := parseWarpStatus(raw)
			if state == "connected" {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("cloudflarewarp: timeout waiting for Connected state")
}

func detectVersion() string {
	binary, err := warpBinary()
	if err != nil {
		return "not installed"
	}
	out, _ := runCmd(binary, "--version")
	return strings.TrimSpace(out)
}
