package forticlient

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

// binaryPath returns the FortiClient CLI binary for the current OS.
func binaryPath() string {
	switch runtime.GOOS {
	case "windows":
		// Default installation path for FortiClient 6.4.x on Windows.
		return `C:\Program Files\Fortinet\FortiClient\FortiClient.exe`
	case "darwin":
		return "/Applications/FortiClient.app/Contents/MacOS/FortiClientAgent"
	default: // linux
		return "forticlientsslvpn"
	}
}

// connectV6 initiates a FortiClient 6.4.x SSL VPN connection.
// Windows: FortiClient.exe /vpnconnect <tunnel_name>
// Linux/macOS: forticlientsslvpn with config file
func connectV6(tunnelName, host string, port int, certPath, keyPath, username, password string) error {
	binary := binaryPath()

	switch runtime.GOOS {
	case "windows":
		_, err := runCmd(binary,
			"/vpnconnect", tunnelName,
		)
		return err

	case "darwin", "linux":
		// FortiClient SSL VPN on Linux/macOS requires a config file.
		cfgPath, err := writeTempConfig(host, port, certPath, keyPath, username, password)
		if err != nil {
			return fmt.Errorf("forticlient: write config: %w", err)
		}
		_, err = runCmdBackground(binary, "--vpnconfig", cfgPath)
		return err

	default:
		return fmt.Errorf("forticlient: unsupported OS %q", runtime.GOOS)
	}
}

// disconnectV6 disconnects a FortiClient 6.4.x SSL VPN tunnel.
func disconnectV6(tunnelName string) error {
	binary := binaryPath()
	switch runtime.GOOS {
	case "windows":
		_, err := runCmd(binary, "/vpndisconnect", tunnelName)
		return err
	default:
		// On Linux/macOS, kill the forticlientsslvpn process.
		_, err := runCmd("pkill", "-f", "forticlientsslvpn")
		return err
	}
}

// detectTunnelInterface polls network interfaces until a FortiClient
// tunnel interface appears (typically named "fortissl" or "vpnssl0").
func detectTunnelInterface(timeout time.Duration) (string, net.IP, error) {
	deadline := time.Now().Add(timeout)
	candidates := []string{"fortissl", "vpnssl0", "ssl0", "ppp0"}

	for time.Now().Before(deadline) {
		ifaces, err := net.Interfaces()
		if err != nil {
			return "", nil, err
		}
		for _, iface := range ifaces {
			name := strings.ToLower(iface.Name)
			for _, c := range candidates {
				if strings.HasPrefix(name, c) {
					addrs, _ := iface.Addrs()
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
							return iface.Name, ipnet.IP, nil
						}
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", nil, fmt.Errorf("forticlient: tunnel interface not found within %s", timeout)
}
