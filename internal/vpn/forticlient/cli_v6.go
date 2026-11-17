package forticlient

import (
	"context"
	"fmt"
	"runtime"
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
func tryGUIConnect(ctx context.Context, tunnelName, username, password string, allowInsecureCert bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("gui automation not supported on %s", runtime.GOOS)
	}
	return guiConnect(ctx, tunnelName, username, password, allowInsecureCert)
}

// tryGUIDisconnect attempts GUI automation to disconnect. Non-Windows returns error.
func tryGUIDisconnect(tunnelName string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("gui automation not supported on %s", runtime.GOOS)
	}
	return guiDisconnect(tunnelName)
}
