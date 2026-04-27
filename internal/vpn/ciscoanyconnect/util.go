package ciscoanyconnect

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// runCmdWithStdin sends lines of input to a command's stdin and returns stdout+stderr.
// Used for interactive VPN CLIs (AnyConnect, GlobalProtect) that prompt for credentials.
func runCmdWithStdin(name string, args []string, input string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// checkServiceRunning verifies the AnyConnect VPN agent is running.
// Returns an error with a helpful message if the service is down.
func checkServiceRunning() error {
	// Try a lightweight status call; if the agent is down it exits non-zero or hangs briefly.
	out, err := runCmd(binaryPath(), "state")
	if err != nil && strings.Contains(strings.ToLower(out+err.Error()), "service") {
		return fmt.Errorf("ciscoanyconnect: VPN agent not running — start the vpnagentd service and retry")
	}
	return nil
}
