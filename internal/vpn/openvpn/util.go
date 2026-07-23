package openvpn

import (
	"bytes"
	"os/exec"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// runCmd executes a command and returns combined stdout+stderr output.
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
