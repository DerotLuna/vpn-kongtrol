package tailscale

import (
	"bytes"
	"os/exec"
	"runtime"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func binaryPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\Tailscale\tailscale.exe`
	}
	return "tailscale"
}
