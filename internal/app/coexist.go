package app

import (
	"bufio"
	"errors"
	"os"
	"slices"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ErrFullTunnelConflict marks a connect refused because another VPN already
// carries all traffic (ADR-0099). Callers that retry on their own (the
// watchdog, the scheduler, traffic-triggered connects) can test it with
// errors.Is and wait for the user instead of retrying.
var ErrFullTunnelConflict = errors.New("another VPN already carries all traffic")

// ExternalTunnel is a provider app's tunnel that is up outside Kongtrol
// (found by the provider's interface matcher, appattach.ActiveTunnel).
type ExternalTunnel struct {
	Type      string // adapter type of the provider (protonvpn, nordvpn, …)
	Label     string // name shown to the user (Proton VPN)
	Interface string
}

// TunnelClaim is a tunnel that already carries all traffic.
type TunnelClaim struct {
	Profile   string // Kongtrol profile; "" for an app outside Kongtrol
	Label     string // what the user sees: the profile name or the app name
	Interface string
}

// fullTunnelApps are the provider apps whose own tunnel sends every
// destination through the VPN by default (consumer VPNs). Corporate
// clients (FortiClient, Cisco, GlobalProtect) are usually split tunnels
// configured by the company, so they are not counted.
var fullTunnelApps = map[string]bool{"protonvpn": true, "nordvpn": true, "surfshark": true}

// FullTunnelAppTypes lists the provider types whose app tunnel counts as a
// full tunnel (for the daemon's external tunnel probe).
func FullTunnelAppTypes() []string {
	out := make([]string, 0, len(fullTunnelApps))
	for t := range fullTunnelApps {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// ProfileCarriesAllTraffic reports, before connecting, whether profile v
// will take the whole default route (0.0.0.0/0, ::/0 or both /1 halves):
//   - consumer provider apps (Proton, NordVPN, Surfshark) by default;
//   - a direct WireGuard profile only with default_egress: the daemon
//     replaces AllowedIPs with the rule destinations and the tunnel's own
//     subnet otherwise (ProfileService.policyAllowedIPs);
//   - a direct OpenVPN profile whose .ovpn has redirect-gateway (a route
//     pushed by the server is only known once connected: RouteReporter).
func ProfileCarriesAllTraffic(v config.VPNConfig) bool {
	switch {
	case fullTunnelApps[v.Type]:
		return true
	case v.Type == "wireguard":
		return v.DefaultEgress
	case v.Type == "openvpn":
		return v.DefaultEgress || ovpnRedirectsGateway(v.ConfigFile)
	}
	return false
}

// ovpnRedirectsGateway reports whether an .ovpn asks for the default route
// itself (redirect-gateway / redirect-def1 directive, not a comment).
func ovpnRedirectsGateway(path string) bool {
	if path == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(strings.TrimPrefix(fields[0], "--")) {
		case "redirect-gateway", "redirect-def1":
			return true
		}
	}
	return false
}

// fullTunnelConflicts decides whether connecting candidate would start a
// second tunnel that takes all traffic. candidateType is its adapter type
// and candidateFull whether it carries all traffic. managed are the
// Kongtrol profiles connected now that carry all traffic, managedIfaces
// the interfaces of every connected Kongtrol profile (a Proton .conf named
// "proton-ch" gives the direct tunnel an interface the Proton matcher
// would also accept), external the provider app tunnels seen up.
//
// A split tunnel never conflicts: its routes are narrower than the full
// tunnel's, so both work (WireGuard, Tailscale and Cloudflare document
// split routes as the way to run two clients). Connecting a provider
// profile while that provider's own app is already up is the attach case,
// not a conflict.
func fullTunnelConflicts(candidate, candidateType string, candidateFull bool, managed []TunnelClaim, managedIfaces []string, external []ExternalTunnel) []TunnelClaim {
	if !candidateFull {
		return nil
	}
	var out []TunnelClaim
	for _, m := range managed {
		if m.Profile != candidate {
			out = append(out, m)
		}
	}
	for _, e := range external {
		if e.Type == candidateType {
			continue
		}
		if e.Interface != "" && slices.ContainsFunc(managedIfaces, func(i string) bool { return strings.EqualFold(i, e.Interface) }) {
			continue
		}
		out = append(out, TunnelClaim{Label: e.Label, Interface: e.Interface})
	}
	return out
}

// FullTunnelConflictError is the translated refusal of a connect that would
// start a second full tunnel, or (ADR-0103) a direct Proton profile next to
// Proton's app or a second tunnel with the same address; it wraps
// ErrFullTunnelConflict.
type FullTunnelConflictError struct {
	*config.ValidationError
	Profile string
	Other   TunnelClaim
}

func (e *FullTunnelConflictError) Unwrap() []error {
	return []error{e.ValidationError, ErrFullTunnelConflict}
}

func newFullTunnelConflictError(profile string, other TunnelClaim) *FullTunnelConflictError {
	key := "api.err.connect_full_conflict"
	if other.Profile == "" {
		key = "api.err.connect_full_conflict_app"
	}
	return &FullTunnelConflictError{
		ValidationError: &config.ValidationError{Key: key, Args: []any{profile, other.Label}},
		Profile:         profile,
		Other:           other,
	}
}

// managedFullTunnels returns the connected Kongtrol profiles (other than
// skip) that carry all traffic, and the interfaces of every connected one.
func managedFullTunnels(cfg *config.Config, adapters *vpn.Set, skip string) (full []TunnelClaim, ifaces []string) {
	for name, a := range adapters.Snapshot() {
		if name == skip || a.Status().Normalize() != vpn.StatusConnected {
			continue
		}
		iface := ""
		if info, err := a.TunnelInfo(); err == nil && info != nil {
			iface = info.InterfaceName
		}
		if iface != "" {
			ifaces = append(ifaces, iface)
		}
		all := false
		if cfg != nil {
			all = ProfileCarriesAllTraffic(cfg.VPNs[name])
		}
		if rr, ok := a.(vpn.RouteReporter); ok {
			if tr := rr.TunnelRoutes(); tr != nil && tr.AllTraffic {
				all = true
			}
		}
		if all {
			full = append(full, TunnelClaim{Profile: name, Label: name, Interface: iface})
		}
	}
	slices.SortFunc(full, func(a, b TunnelClaim) int { return strings.Compare(a.Profile, b.Profile) })
	return full, ifaces
}
