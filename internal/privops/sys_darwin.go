//go:build darwin

package privops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const runDir = "/var/run"

func configureOVPNCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// ── Routes (/sbin/route, as internal/routing/darwin.go) ──────────────────

func routeArgs(verb string, r Route) []string {
	args := []string{verb}
	if r.Destination.Addr().Is6() {
		args = append(args, "-inet6")
	}
	args = append(args, "-net", r.Destination.String())
	if r.Gateway.IsValid() {
		args = append(args, r.Gateway.String())
	} else if verb == "add" {
		args = append(args, "-interface", r.Interface)
	}
	return args
}

func sysAddRoute(ctx context.Context, r Route) error {
	_, err := run(ctx, "/sbin/route", routeArgs("add", r)...)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "file exists") {
		return nil
	}
	return err
}

func sysDeleteRoute(ctx context.Context, r Route) error {
	_, err := run(ctx, "/sbin/route", routeArgs("delete", r)...)
	return err
}

// ── DNS (networksetup, resolv.conf fallback) ────────────────────────────

const (
	networksetup = "/usr/sbin/networksetup"
	resolvConf   = "/etc/resolv.conf"
)

type dnsSnapshot struct {
	service string // network service name, or "" when resolv.conf was used
	content []byte
}

func darwinService(ctx context.Context, iface string) (string, error) {
	out, err := run(ctx, networksetup, "-listallhardwareports")
	if err != nil {
		return "", err
	}
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "Device: "+iface {
			for j := i - 1; j >= 0; j-- {
				if strings.HasPrefix(lines[j], "Hardware Port:") {
					return strings.TrimSpace(strings.TrimPrefix(lines[j], "Hardware Port:")), nil
				}
			}
		}
	}
	return "", errNoDarwinService
}

func sysSnapshotDNS(ctx context.Context, iface string) (dnsSnapshot, error) {
	if svc, err := darwinService(ctx, iface); err == nil {
		return dnsSnapshot{service: svc}, nil
	}
	b, err := os.ReadFile(resolvConf)
	if err != nil {
		return dnsSnapshot{}, err
	}
	return dnsSnapshot{content: b}, nil
}

func sysSetDNS(ctx context.Context, req DNSRequest) error {
	if svc, err := darwinService(ctx, req.Interface); err == nil {
		args := []string{"-setdnsservers", svc}
		for _, s := range req.Servers {
			args = append(args, s.String())
		}
		_, err := run(ctx, networksetup, args...)
		return err
	}
	var sb strings.Builder
	sb.WriteString("# Managed by vpn-kongtrol\n")
	for _, s := range req.Servers {
		sb.WriteString("nameserver " + s.String() + "\n")
	}
	return os.WriteFile(resolvConf, []byte(sb.String()), 0o644)
}

func sysRestoreDNS(ctx context.Context, _ string, snap dnsSnapshot) error {
	if snap.service != "" {
		_, err := run(ctx, networksetup, "-setdnsservers", snap.service, "empty")
		return err
	}
	if snap.content != nil {
		return os.WriteFile(resolvConf, snap.content, 0o644)
	}
	return nil
}

// ── Kill switch (pf, anchor "com.kongtrol") ─────────────────────────────
//
// As other VPN clients do: the system ruleset (/etc/pf.conf with Apple's
// anchors) is reloaded with `anchor "com.kongtrol"` appended, the rules go
// into that anchor, and pf is enabled with a reference token (pfctl -E).
// Turning off flushes the anchor, reloads /etc/pf.conf and releases the
// token (pfctl -X), never `pfctl -d`, so pf stays on for anyone else.

const (
	pfctl        = "/sbin/pfctl"
	pfConf       = "/etc/pf.conf"
	legacyAnchor = "kongtrol" // anchor and file used by earlier releases
	legacyFile   = "/etc/pf.anchors/kongtrol"
)

// pfState is guarded by Local.mu (every kill switch call holds it).
var pfState struct {
	token      string // from pfctl -E, released by pfctl -X
	loadedMain bool   // the loaded main ruleset references our anchor
}

// runInput is run with input on stdin (pfctl -f -), so no rules file is
// ever written where another user could reach it.
func runInput(ctx context.Context, input []byte, exe string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = safeEnv()
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s", filepath.Base(exe), strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func readPFConf() ([]byte, error) {
	f, err := os.Open(pfConf)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxPFConf+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxPFConf {
		return nil, fmt.Errorf("%s is larger than %d bytes", pfConf, maxPFConf)
	}
	return b, nil
}

func sysKillSwitchEnable(ctx context.Context, req KillSwitchRequest) error {
	rules, err := pfKillSwitchRules(req)
	if err != nil {
		return err
	}
	base, err := readPFConf()
	if err != nil {
		return err
	}
	if _, err := runInput(ctx, pfMainRuleset(base), pfctl, "-f", "-"); err != nil {
		return err
	}
	pfState.loadedMain = true
	if _, err := runInput(ctx, []byte(rules), pfctl, "-a", pfAnchor, "-f", "-"); err != nil {
		_ = sysKillSwitchDisable(ctx)
		return err
	}
	out, err := run(ctx, pfctl, "-E")
	if err != nil {
		_ = sysKillSwitchDisable(ctx)
		return err
	}
	tok, err := parsePFToken(out)
	if err != nil {
		_ = sysKillSwitchDisable(ctx)
		return err
	}
	pfState.token = tok
	return nil
}

// sysKillSwitchDisable always empties our anchor (harmless when it does not
// exist). The system ruleset is reloaded and the token released only when
// this process changed them, so a ruleset loaded by another tool is never
// replaced. After a helper restart the main ruleset may still reference
// the anchor, which is then empty and filters nothing.
func sysKillSwitchDisable(ctx context.Context) error {
	var errs []error
	if _, err := run(ctx, pfctl, "-a", pfAnchor, "-F", "all"); err != nil && pfState.loadedMain {
		errs = append(errs, err)
	}
	if pfState.loadedMain {
		if _, err := run(ctx, pfctl, "-f", pfConf); err != nil {
			errs = append(errs, err)
		} else {
			pfState.loadedMain = false
		}
	}
	if tok := pfState.token; tok != "" {
		if _, err := run(ctx, pfctl, "-X", tok); err != nil {
			errs = append(errs, err)
		} else {
			pfState.token = ""
		}
	}
	if _, err := os.Lstat(legacyFile); err == nil {
		_, _ = run(ctx, pfctl, "-a", legacyAnchor, "-F", "all")
		_ = os.Remove(legacyFile)
	}
	return errors.Join(errs...)
}

// sysWGIsTunnel reports whether name is a wg-quick/wireguard-go tunnel
// (root-owned /var/run/wireguard/<name>.name maps it to a utun). An
// interface of that name without it is denied.
func sysWGIsTunnel(_ context.Context, name string) (bool, error) {
	if rootOwnedFile("/var/run/wireguard/"+name+".name", false) {
		return true, nil
	}
	if _, err := net.InterfaceByName(name); err == nil {
		return false, denied(DenyNotWireGuard, "interface %q is not a WireGuard tunnel", name)
	}
	return false, nil
}
