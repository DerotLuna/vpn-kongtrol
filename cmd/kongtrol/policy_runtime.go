package main

import (
	"net/netip"
	"path/filepath"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// tunnelOfLocalAddr names the connected profile whose tunnel has local
// address a ("" = not a Kongtrol tunnel): a connection with that source
// address left through that tunnel (recent destinations, ADR-0054).
func tunnelOfLocalAddr(a netip.Addr) string {
	if col == nil || !a.IsValid() {
		return ""
	}
	for name, m := range col.Snapshot() {
		if m.Status.Normalize() != vpn.StatusConnected || m.AssignedIP == "" {
			continue
		}
		if ip, err := netip.ParseAddr(strings.TrimSpace(m.AssignedIP)); err == nil && ip.Unmap() == a.Unmap() {
			return name
		}
	}
	return ""
}

// probeTunnel reports a profile's tunnel to the policy resolver, which
// follows tunnels that come up or drop outside Connect/Disconnect
// (provider apps, watchdog reconnects).
func probeTunnel(profile string) monitor.TunnelState {
	a, ok := adapters.Get(profile)
	if !ok || cfg == nil {
		return monitor.TunnelState{}
	}
	st := monitor.TunnelState{Type: cfg.VPNs[profile].Type, ConfigFile: cfg.VPNs[profile].ConfigFile}
	if a.Status().Normalize() != vpn.StatusConnected {
		return st
	}
	if policyWatch != nil {
		policyWatch.ResetConnectBudget(profile)
	}
	st.Connected = true
	if info, err := a.TunnelInfo(); err == nil && info != nil {
		st.Interface, st.DNS = info.InterfaceName, info.DNS
	}
	if st.Type == "wireguard" {
		st.Interface = interfaceOfWireGuardProfile(profile)
	}
	return st
}

// interfaceOfWireGuardProfile mirrors app.interfaceFromWGConfig: the
// tunnel is named after tunnel_name or the config file.
func interfaceOfWireGuardProfile(profile string) string {
	v := cfg.VPNs[profile]
	if v.TunnelName != "" {
		return v.TunnelName
	}
	base := filepath.Base(v.ConfigFile)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func profileStatus(profile string) vpn.Status {
	if a, ok := adapters.Get(profile); ok {
		return a.Status().Normalize()
	}
	return vpn.StatusDisconnected
}

// logPolicyEvent writes a policy notice to the daemon log and the audit
// log in the daemon's language; clients render the event themselves.
func logPolicyEvent(e monitor.Event) {
	args := make([]any, len(e.Args))
	for i, a := range e.Args {
		args[i] = a
	}
	msg := cf(e.Key, args...)
	emitAlert(e.Level, e.Profile, msg)
	logAudit(e.Level, "policy.tunnel_down."+e.Action, e.Profile, msg)
}
