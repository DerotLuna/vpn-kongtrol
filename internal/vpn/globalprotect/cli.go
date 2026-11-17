// GlobalProtect adapter CLI bindings.
// Supported on Windows and macOS only — Linux has no official GlobalProtect client.
package globalprotect

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

// ErrSSORequired is returned when the portal enforces SSO-only authentication.
var ErrSSORequired = fmt.Errorf("globalprotect: portal requires SSO authentication — connect manually via the GlobalProtect client")

// ErrLinux is returned on unsupported platforms.
var ErrLinux = fmt.Errorf("globalprotect: Palo Alto GlobalProtect has no Linux client — use an alternative VPN adapter")

func binaryPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return `C:\Program Files\Palo Alto Networks\GlobalProtect\globalprotect.exe`, nil
	case "darwin":
		return "/Applications/GlobalProtect.app/Contents/MacOS/globalprotect", nil
	default:
		return "", ErrLinux
	}
}

func connect(host, username, password string) error {
	binary, err := binaryPath()
	if err != nil {
		return err
	}

	args := []string{"connect", "--portal", host}
	if username != "" {
		args = append(args, "--username", username)
	}

	input := ""
	if password != "" {
		input = password + "\ny\n"
	}

	var out string
	if input != "" {
		out, err = runCmdWithStdin(binary, args, input)
	} else {
		out, err = runCmd(binary, args...)
	}

	lower := strings.ToLower(out)
	if strings.Contains(lower, "authentication method not supported") {
		return ErrSSORequired
	}
	if err != nil {
		return fmt.Errorf("globalprotect connect: %w\n%s", err, out)
	}
	return nil
}

func disconnect() error {
	binary, err := binaryPath()
	if err != nil {
		return err
	}
	out, err := runCmd(binary, "disconnect")
	if err != nil {
		return fmt.Errorf("globalprotect disconnect: %w\n%s", err, out)
	}
	return nil
}

// detectInterface polls for a GlobalProtect tunnel interface (gpd0).
func detectInterface(timeout time.Duration) (string, net.IP, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			if strings.HasPrefix(strings.ToLower(iface.Name), "gpd") ||
				strings.HasPrefix(strings.ToLower(iface.Name), "pan") {
				addrs, _ := iface.Addrs()
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
						return iface.Name, ipnet.IP, nil
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", nil, fmt.Errorf("globalprotect: tunnel interface not found within %s", timeout)
}

func detectVersion() string {
	binary, err := binaryPath()
	if err != nil {
		return "unsupported"
	}
	out, _ := runCmd(binary, "--version")
	return strings.TrimSpace(out)
}
