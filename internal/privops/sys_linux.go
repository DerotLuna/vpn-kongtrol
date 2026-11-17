//go:build linux

package privops

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
)

const runDir = "/run"

func configureOVPNCmd(cmd *exec.Cmd) {
	// Own process group; die with the helper even if it crashes.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGTERM}
}

// ── Routes (netlink) ────────────────────────────────────────────────────

func nlRoute(r Route) (*netlink.Route, error) {
	link, err := netlink.LinkByName(r.Interface)
	if err != nil {
		return nil, err
	}
	dst := &net.IPNet{IP: r.Destination.Addr().AsSlice(), Mask: net.CIDRMask(r.Destination.Bits(), r.Destination.Addr().BitLen())}
	nr := &netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Priority: r.Metric}
	if r.Gateway.IsValid() {
		nr.Gw = r.Gateway.AsSlice()
	}
	return nr, nil
}

func sysAddRoute(_ context.Context, r Route) error {
	nr, err := nlRoute(r)
	if err != nil {
		return err
	}
	if err := netlink.RouteAdd(nr); err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
		return err
	}
	return nil
}

func sysDeleteRoute(_ context.Context, r Route) error {
	nr, err := nlRoute(r)
	if err != nil {
		return err
	}
	nr.Priority = 0
	return netlink.RouteDel(nr)
}

// ── DNS (/etc/resolv.conf, as internal/security/dnsguard_unix.go) ───────

const (
	resolvConf   = "/etc/resolv.conf"
	resolvBackup = "/etc/resolv.conf.kongtrol.bak"
)

type dnsSnapshot struct{ content []byte }

func sysSnapshotDNS(_ context.Context, _ string) (dnsSnapshot, error) {
	b, err := os.ReadFile(resolvConf)
	if err != nil {
		return dnsSnapshot{}, err
	}
	if err := os.WriteFile(resolvBackup, b, 0o644); err != nil {
		return dnsSnapshot{}, err
	}
	return dnsSnapshot{content: b}, nil
}

func sysSetDNS(_ context.Context, req DNSRequest) error {
	var sb strings.Builder
	sb.WriteString("# Managed by vpn-kongtrol — original backed up at " + resolvBackup + "\n")
	for _, s := range req.Servers {
		sb.WriteString("nameserver " + s.String() + "\n")
	}
	return os.WriteFile(resolvConf, []byte(sb.String()), 0o644)
}

func sysRestoreDNS(_ context.Context, _ string, snap dnsSnapshot) error {
	if err := os.WriteFile(resolvConf, snap.content, 0o644); err != nil {
		return err
	}
	_ = os.Remove(resolvBackup)
	return nil
}

// ── Kill switch (iptables, dedicated KONGTROL chain) ─────────────────────
//
// Unlike internal/security/killswitch_linux.go (which flushes the whole
// OUTPUT chain), the helper only touches its own chain and one jump rule.

const ksChain = "KONGTROL"

func iptables() (string, error) {
	return trustedBinary("/usr/sbin/iptables", "/sbin/iptables", "/usr/bin/iptables")
}

func sysKillSwitchEnable(ctx context.Context, req KillSwitchRequest) error {
	ipt, err := iptables()
	if err != nil {
		return err
	}
	rules := [][]string{
		{"-N", ksChain},
		{"-A", ksChain, "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		{"-A", ksChain, "-o", "lo", "-j", "ACCEPT"},
	}
	for _, i := range req.Interfaces {
		rules = append(rules, []string{"-A", ksChain, "-o", i, "-j", "ACCEPT"})
	}
	for _, e := range req.Endpoints {
		if e.Is4() {
			rules = append(rules, []string{"-A", ksChain, "-d", e.String(), "-j", "ACCEPT"})
		}
	}
	if req.AllowLAN {
		for _, s := range []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"} {
			rules = append(rules, []string{"-A", ksChain, "-d", s, "-j", "ACCEPT"})
		}
	}
	rules = append(rules, []string{"-A", ksChain, "-j", "DROP"}, []string{"-I", "OUTPUT", "1", "-j", ksChain})
	for _, r := range rules {
		if _, err := run(ctx, ipt, r...); err != nil {
			_ = sysKillSwitchDisable(ctx)
			return err
		}
	}
	return nil
}

func sysKillSwitchDisable(ctx context.Context) error {
	ipt, err := iptables()
	if err != nil {
		return err
	}
	for i := 0; i < 4; i++ { // remove every jump we may have inserted
		if _, err := run(ctx, ipt, "-D", "OUTPUT", "-j", ksChain); err != nil {
			break
		}
	}
	_, _ = run(ctx, ipt, "-F", ksChain)
	_, _ = run(ctx, ipt, "-X", ksChain)
	return nil
}

// sysWGIsTunnel reports whether name is a WireGuard interface (kernel
// "wireguard" link, or a wireguard-go tun with its root-owned control
// socket); another interface of that name is denied.
func sysWGIsTunnel(_ context.Context, name string) (bool, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		var nf netlink.LinkNotFoundError
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	if link.Type() == "wireguard" || rootOwnedFile("/var/run/wireguard/"+name+".sock", true) {
		return true, nil
	}
	return false, denied(DenyNotWireGuard, "interface %q is not a WireGuard tunnel", name)
}
