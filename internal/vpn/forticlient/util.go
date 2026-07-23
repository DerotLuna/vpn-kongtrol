package forticlient

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// runCmd executes a command synchronously and returns combined output.
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// runCmdBackground starts a command without waiting for it to finish.
func runCmdBackground(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	vpn.HideChildWindow(cmd)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start %s: %w", name, err)
	}
	return "", nil
}

// writeTempConfig writes a FortiClient SSL VPN config file to a temp location.
// Used on Linux/macOS where FortiClient accepts a config file argument.
const configTemplate = `
[vpn]
server={{.Host}}:{{.Port}}
username={{.Username}}
cert={{.Cert}}
key={{.Key}}
`

type configData struct {
	Host     string
	Port     int
	Username string
	Cert     string
	Key      string
}

// ifaceByteCounters reads sent/received byte counts for a network interface.
// On Windows uses PowerShell Get-NetAdapterStatistics; on Linux reads /sys/class/net.
func ifaceByteCounters(name string) (sent, received uint64) {
	switch runtime.GOOS {
	case "windows":
		// Get-NetAdapterStatistics returns SentBytes and ReceivedBytes.
		ps := fmt.Sprintf(
			`$s = Get-NetAdapterStatistics -Name '%s' -ErrorAction SilentlyContinue; `+
				`if ($s) { "$($s.SentBytes) $($s.ReceivedBytes)" }`, name)
		out, err := runCmd("powershell", "-NoProfile", "-Command", ps)
		if err != nil {
			return 0, 0
		}
		parts := strings.Fields(strings.TrimSpace(out))
		if len(parts) == 2 {
			sent, _ = strconv.ParseUint(parts[0], 10, 64)
			received, _ = strconv.ParseUint(parts[1], 10, 64)
		}
	case "linux":
		readSys := func(path string) uint64 {
			data, err := os.ReadFile(path)
			if err != nil {
				return 0
			}
			v, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
			return v
		}
		sent = readSys("/sys/class/net/" + name + "/statistics/tx_bytes")
		received = readSys("/sys/class/net/" + name + "/statistics/rx_bytes")
	}
	return
}

// ifaceDNSServers reads the DNS servers assigned to a network interface.
// FortiClient pushes DNS servers through the tunnel; this reads them from the OS.
func ifaceDNSServers(name string) []net.IP {
	switch runtime.GOOS {
	case "windows":
		ps := fmt.Sprintf(
			`(Get-DnsClientServerAddress -InterfaceAlias '%s' -AddressFamily IPv4 -ErrorAction SilentlyContinue).ServerAddresses -join ' '`, name)
		out, err := runCmd("powershell", "-NoProfile", "-Command", ps)
		if err != nil {
			return nil
		}
		var servers []net.IP
		for _, s := range strings.Fields(strings.TrimSpace(out)) {
			if ip := net.ParseIP(s); ip != nil {
				servers = append(servers, ip)
			}
		}
		return servers
	case "linux", "darwin":
		// FortiClient on Linux/macOS typically modifies /etc/resolv.conf.
		// Read nameservers from the interface's DNS config.
		data, err := os.ReadFile("/etc/resolv.conf")
		if err != nil {
			return nil
		}
		var servers []net.IP
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "nameserver ") {
				if ip := net.ParseIP(strings.TrimPrefix(line, "nameserver ")); ip != nil {
					servers = append(servers, ip)
				}
			}
		}
		return servers
	}
	return nil
}

func writeTempConfig(host string, port int, cert, key, username, _ string) (string, error) {
	tmpl, err := template.New("cfg").Parse(configTemplate)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "kongtrol-forti-*")
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, "forti.conf")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return path, tmpl.Execute(f, configData{
		Host:     host,
		Port:     port,
		Username: username,
		Cert:     cert,
		Key:      key,
	})
}
