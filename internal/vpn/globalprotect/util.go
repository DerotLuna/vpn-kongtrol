package globalprotect

import (
	"bytes"
	"os/exec"
	"strings"

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

func runCmdWithStdin(name string, args []string, input string) (string, error) {
	cmd := exec.Command(name, args...)
	vpn.HideChildWindow(cmd)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
