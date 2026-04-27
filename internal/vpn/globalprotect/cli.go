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

func statusOutput() (string, error) {
	binary, err := binaryPath()
	if err != nil {
		return "", err
	}
	return runCmd(binary, "status")
}

// parseStatus parses `globalprotect status` output.
// Returns (connected, assignedIP, gateway).
func parseStatus(raw string) (connected bool, assignedIP, gateway string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "status:") {
			connected = strings.Contains(lower, "connected")
		}
		if strings.HasPrefix(lower, "assigned-ip:") || strings.HasPrefix(lower, "assigned ip:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				assignedIP = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(lower, "gateway:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				gateway = strings.TrimSpace(parts[1])
			}
		}
	}
	return
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
