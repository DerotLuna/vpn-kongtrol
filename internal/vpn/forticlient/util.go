package forticlient

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// runCmd executes a command synchronously and returns combined output.
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// runCmdBackground starts a command without waiting for it to finish.
func runCmdBackground(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
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
