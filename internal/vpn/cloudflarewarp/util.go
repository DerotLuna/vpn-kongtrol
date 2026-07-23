package cloudflarewarp

import (
	"bytes"
	"fmt"
	"os"
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

// warpBinary returns the path to warp-cli, checking common install locations.
func warpBinary() (string, error) {
	candidates := []string{"warp-cli"}
	switch runtime.GOOS {
	case "windows":
		candidates = append(candidates,
			`C:\Program Files\Cloudflare\Cloudflare WARP\warp-cli.exe`,
		)
	case "darwin":
		candidates = append(candidates, "/Applications/Cloudflare WARP.app/Contents/MacOS/warp-cli")
	case "linux":
		candidates = append(candidates, "/usr/bin/warp-cli", "/usr/local/bin/warp-cli")
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("cloudflarewarp: warp-cli not found — install Cloudflare WARP first")
}
