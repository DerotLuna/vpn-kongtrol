package tailscale

import (
	"bytes"
	"os/exec"
	"runtime"
)

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
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
