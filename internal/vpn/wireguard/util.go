package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const cmdTimeout = 90 * time.Second

func runCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return out.String(), fmt.Errorf("%s timed out after %s", name, cmdTimeout)
	}
	return out.String(), err
}
