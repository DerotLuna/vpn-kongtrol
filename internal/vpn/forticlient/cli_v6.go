package forticlient

import (
	"context"
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

// versionCmd returns the FortiClient version without launching the GUI.
// On Windows, reads the file version via PowerShell to avoid blocking.
func versionCmd() (string, error) {
	if runtime.GOOS == "windows" {
		return runCmd("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(`(Get-Item '%s').VersionInfo.ProductVersion`, binaryPath()))
	}
	return runCmd(binaryPath(), "--version")
}

// connectV6 initiates a FortiClient 6.4.x SSL VPN connection.
// Windows: FortiClient.exe /vpnconnect <tunnel_name>
// Linux/macOS: forticlientsslvpn with config file
func connectV6(tunnelName, host string, port int, certPath, keyPath, username, password string) error {
	binary := binaryPath()

	switch runtime.GOOS {
	case "windows":
		// Launch in background — FortiClient.exe is a GUI app that blocks
		// if launched synchronously. Tunnel detection happens in the caller.
		_, err := runCmdBackground(binary,
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
		// Launch in background — FortiClient.exe is a GUI app that blocks.
		_, err := runCmdBackground(binary, "/vpndisconnect", tunnelName)
		return err
	default:
		// On Linux/macOS, kill the forticlientsslvpn process.
		_, err := runCmd("pkill", "-f", "forticlientsslvpn")
		return err
	}
}

// tryGUIConnect attempts GUI automation to connect a FortiClient tunnel.
// Returns an error on non-Windows (caller falls back to CLI).
// Implemented in gui_windows.go on Windows; this stub covers other OSes.
func tryGUIConnect(tunnelName, username, password string, allowInsecureCert bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("gui automation not supported on %s", runtime.GOOS)
	}
	return guiConnect(tunnelName, username, password, allowInsecureCert)
}

// tryGUIDisconnect attempts GUI automation to disconnect. Non-Windows returns error.
func tryGUIDisconnect(tunnelName string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("gui automation not supported on %s", runtime.GOOS)
	}
	return guiDisconnect(tunnelName)
}

// disableFortiAdapter disables a Fortinet network adapter as a fallback when
// CLI /vpndisconnect is blocked by EMS policy. On Windows, uses PowerShell
// Disable-NetAdapter which requires admin privileges but reliably drops the tunnel.
// On Linux/macOS, falls back to ifconfig down.
func disableFortiAdapter(ifaceName string) error {
	switch runtime.GOOS {
	case "windows":
		// Disable-NetAdapter by name drops the tunnel without killing FortiClient.
		_, err := runCmd("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(`Disable-NetAdapter -Name '%s' -Confirm:$false`, ifaceName))
		return err
	default:
		_, err := runCmd("ifconfig", ifaceName, "down")
		return err
	}
}

// detectTunnelInterface polls network interfaces until a FortiClient
// tunnel interface appears. ctx cancellation (e.g. Ctrl+C) stops the loop early.
func detectTunnelInterface(ctx context.Context, timeout time.Duration) (string, net.IP, error) {
	if runtime.GOOS == "windows" {
		return detectTunnelInterfaceWindows(ctx, timeout)
	}
	return detectTunnelInterfaceUnix(ctx, timeout)
}

// detectTunnelInterfaceUnix matches by interface name prefix (Linux/macOS).
func detectTunnelInterfaceUnix(ctx context.Context, timeout time.Duration) (string, net.IP, error) {
	deadline := time.Now().Add(timeout)
	candidates := []string{"fortissl", "vpnssl0", "ssl0", "ppp0"}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}
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

// detectTunnelInterfaceWindows uses PowerShell to find Fortinet adapters by
// description, then reads their IP via the standard net package.
// FortiClient creates adapters with descriptions like "Fortinet SSL VPN Virtual
// Ethernet Adapter" or "Fortinet Virtual Ethernet Adapter (NDIS 6.30)".
func detectTunnelInterfaceWindows(ctx context.Context, timeout time.Duration) (string, net.IP, error) {
	deadline := time.Now().Add(timeout)

	// PowerShell command to find a Fortinet adapter that is Up and has an IPv4 address.
	psCmd := `Get-NetAdapter | Where-Object { $_.InterfaceDescription -like '*Fortinet*' -and $_.Status -eq 'Up' } | ` +
		`Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | ` +
		`Select-Object -First 1 -ExpandProperty InterfaceAlias`

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}
		out, err := runCmd("powershell", "-NoProfile", "-Command", psCmd)
		if err == nil {
			name := strings.TrimSpace(out)
			if name != "" {
				// Resolve IP via Go's net package for consistency.
				iface, err := net.InterfaceByName(name)
				if err == nil {
					addrs, _ := iface.Addrs()
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
							return name, ipnet.IP, nil
						}
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return "", nil, fmt.Errorf("forticlient: tunnel interface not found within %s", timeout)
}
