package forticlient

import (
	"fmt"
	"runtime"
)

// defaultWindowsBinary is FortiClient's default install path on Windows
// (DetectInstall prefers app_path and the installer's INSTALLDIR).
const defaultWindowsBinary = `C:\Program Files\Fortinet\FortiClient\FortiClient.exe`

// binaryPath returns the FortiClient binary for the current OS.
func binaryPath() string {
	switch runtime.GOOS {
	case "windows":
		return defaultWindowsBinary
	case "darwin":
		return "/Applications/FortiClient.app/Contents/MacOS/FortiClientAgent"
	default: // linux
		return "forticlientsslvpn"
	}
}

// connectV6 starts a FortiClient SSL VPN connection on Linux / macOS
// (forticlientsslvpn with a config file). Windows uses GUI automation or
// FortiVPN.exe (see SelectPlan).
func connectV6(binary, host string, port int, certPath, keyPath, username, password string) error {
	switch runtime.GOOS {
	case "darwin", "linux":
		cfgPath, err := writeTempConfig(host, port, certPath, keyPath, username, password)
		if err != nil {
			return fmt.Errorf("forticlient: write config: %w", err)
		}
		_, err = runCmdBackground(binary, "--vpnconfig", cfgPath)
		return err
	default:
		return fmt.Errorf("forticlient: no CLI connect on %s", runtime.GOOS)
	}
}

// disconnectV6 disconnects a FortiClient SSL VPN tunnel: on Windows
// `FortiClient.exe /vpndisconnect <tunnel>` (started in the background: the
// GUI app blocks), elsewhere by stopping forticlientsslvpn.
func disconnectV6(binary, tunnelName string) error {
	switch runtime.GOOS {
	case "windows":
		_, err := runCmdBackground(binary, "/vpndisconnect", tunnelName)
		return err
	default:
		_, err := runCmd("pkill", "-f", "forticlientsslvpn")
		return err
	}
}
