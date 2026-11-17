package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// helperInstallMu serializes GUI installs: a second click waits for the
// first, then finds the helper answering and returns at once.
var helperInstallMu sync.Mutex

// installHelperFromGUI backs POST /api/v1/helper/install: it runs this
// executable as `service install --quiet` with no console window and no
// terminal, so the child asks for permission with the OS dialog (UAC,
// osascript, pkexec). Errors are translated; a declined prompt wraps
// privhelper.ErrElevationCancelled.
func installHelperFromGUI(ctx context.Context) error {
	helperInstallMu.Lock()
	defer helperInstallMu.Unlock()

	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	res, err := privops.NewClient().Ping(pctx)
	cancel()
	if err == nil && res.Protocol == privops.ProtocolVersion {
		return nil
	}

	exe, err := currentExecutable()
	if err != nil {
		return errors.New(cf("service.install.failed_detail", err))
	}
	cmd := exec.CommandContext(ctx, exe, "service", "install", "--quiet")
	// English output so the cancelled line below can be recognised.
	cmd.Env = append(os.Environ(), "KONGTROL_LANG=en", "NO_COLOR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	vpn.HideChildWindow(cmd)
	if err := cmd.Run(); err != nil {
		out := stderr.String()
		switch {
		case strings.Contains(out, i18n.T(i18n.EN, "service.elevation.cancelled")):
			return fmt.Errorf("%s: %w", ct("service.elevation.cancelled"), privhelper.ErrElevationCancelled)
		case ctx.Err() != nil:
			return fmt.Errorf("%s: %w", ct("service.install.failed"), ctx.Err())
		}
		return errors.New(cf("service.install.failed_detail", firstLine(out, err)))
	}
	return nil
}

// firstLine returns the first non-empty line of out, else err's text.
func firstLine(out string, err error) string {
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return err.Error()
}
