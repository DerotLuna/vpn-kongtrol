//go:build windows

package forticlient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// fortiVPNDisconnectTimeout bounds `FortiVPN.exe --cli --disconnect`
	// (whether it returns promptly is not verified).
	fortiVPNDisconnectTimeout = 15 * time.Second
	// tunnelPoll is how often fortiVPNConnect checks whether the tunnel
	// came up while the auth-dialog script waits.
	tunnelPoll = time.Second
)

// guiConnect automates the FortiClient 6.4 window (StrategyGUI64): select
// the tunnel, fill the credentials and click Connect. Cancelling ctx (the
// user ran `down`, or the attempt timed out) stops the script at once; the
// clipboard is then cleared because the script may have been stopped while
// it held the password.
func guiConnect(ctx context.Context, binary, tunnel, username, password string, allowInsecureCert bool) error {
	out, err := runGuiScript(ctx, "fc_connect.ps1", buildGuiScript(actionConnect, allowInsecureCert),
		scriptInputs(binary, tunnel, username, password))
	if ctx.Err() != nil {
		clearClipboard()
		return ctx.Err()
	}
	return scriptResult(actionConnect, out, err)
}

// guiDisconnect clicks Disconnect in the FortiClient 6.4 window (it
// replaces Connect at the same position).
func guiDisconnect(ctx context.Context, binary string) error {
	out, err := runGuiScript(ctx, "fc_disconnect.ps1", buildGuiScript(actionDisconnect, false),
		scriptInputs(binary, "", "", ""))
	return scriptResult(actionDisconnect, out, err)
}

// fortiVPNConnect (StrategyFortiVPN, from PR #8) starts
// `FortiVPN.exe --cli --connect --tunnel <name>` and, when Kongtrol has
// credentials, fills the native authentication dialog it opens. The
// credentials never appear on a command line. The dialog script stops as
// soon as up() reports the tunnel (saved credentials, SSO): no dialog to
// fill then.
func fortiVPNConnect(ctx context.Context, in Install, tunnel, username, password string, allowInsecureCert bool, up func() bool) error {
	if tunnel == "" {
		return errors.New("forticlient: tunnel_name is empty")
	}
	if _, err := runCmdBackground(in.FortiVPN, fortiVPNConnectArgs(tunnel)...); err != nil {
		return fmt.Errorf("forticlient: FortiVPN.exe: %w", err)
	}
	if username == "" && password == "" {
		return nil // the user answers the dialog; attach mode waits
	}
	sctx, stop := context.WithCancel(ctx)
	defer stop()
	tunnelUp := make(chan struct{})
	go func() {
		t := time.NewTicker(tunnelPoll)
		defer t.Stop()
		for {
			select {
			case <-sctx.Done():
				return
			case <-t.C:
				if up() {
					close(tunnelUp)
					stop()
					return
				}
			}
		}
	}()
	out, err := runGuiScript(sctx, "fc_auth.ps1", buildGuiScript(actionAuth, allowInsecureCert),
		scriptInputs(in.Path, tunnel, username, password))
	select {
	case <-tunnelUp:
		clearClipboard()
		return nil
	default:
	}
	if ctx.Err() != nil {
		clearClipboard()
		return ctx.Err()
	}
	return scriptResult(actionAuth, out, err)
}

// fortiVPNDisconnect runs `FortiVPN.exe --cli --disconnect` (PR #8).
func fortiVPNDisconnect(ctx context.Context, fortiVPN, tunnel string) error {
	ctx, cancel := context.WithTimeout(ctx, fortiVPNDisconnectTimeout)
	defer cancel()
	_, err := runCmdInput(ctx, "", fortiVPN, fortiVPNDisconnectArgs(tunnel)...)
	return err
}

// runGuiScript writes script to a private temp file and runs it with
// Windows PowerShell, passing inputs on stdin (scriptInputs). The file
// starts with a UTF-8 BOM: without one, Windows PowerShell 5.1 reads a
// script in the legacy ANSI code page, which corrupts non-ASCII text
// (dialog titles such as "Sí") and can even break parsing (a UTF-8 em dash
// decodes to a closing smart quote in Windows-1252).
func runGuiScript(ctx context.Context, name, script, inputs string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "kongtrol-fcgui-*")
	if err != nil {
		return "", fmt.Errorf("forticlient gui: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, name)
	if err := writeScriptFile(scriptPath, script); err != nil {
		return "", fmt.Errorf("forticlient gui: write script: %w", err)
	}
	return runCmdInput(ctx, inputs, "powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-File", scriptPath)
}

// writeScriptFile writes a PowerShell script as UTF-8 with BOM, readable
// only by the current user.
func writeScriptFile(path, script string) error {
	return os.WriteFile(path, append([]byte(utf8BOM), script...), 0600)
}

// clearClipboard empties the clipboard after a stopped script, which may
// have been stopped between copying the password and clearing it. Best
// effort.
func clearClipboard() {
	_, _ = runCmd("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::Clear()")
}

// scriptResult turns the script's output into one short error line: the
// script's own "ERROR:" line, else the first line of a PowerShell error
// (see scriptErrorLine), never the whole output.
func scriptResult(action, out string, err error) error {
	if line, ok := scriptReportedError(out); ok {
		return errors.New("forticlient gui: " + line)
	}
	if err != nil {
		return fmt.Errorf("forticlient gui: %s script failed: %s (%w)", action, scriptErrorLine(out), err)
	}
	return nil
}
