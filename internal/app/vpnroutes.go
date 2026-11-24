package app

import (
	"net/netip"
	"slices"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// Routes a VPN brings by itself (ADR-0107): read-only, Kongtrol does not
// own them. Shared by GET /api/v1/vpns/routes and `kongtrol routes vpn`.

// Origin of a VPN's own route.
const (
	OwnRouteFile   = "file"   // route / AllowedIPs line in the .ovpn / .conf
	OwnRouteServer = "server" // pushed by the OpenVPN server (PUSH_REPLY)
)

// Where a profile's own routes come from.
const (
	OwnSourceConfig     = "config"     // read from the profile's file (and the server once connected)
	OwnSourceApp        = "app"        // a provider app decides: Kongtrol cannot see its routes
	OwnSourceUnreadable = "unreadable" // the file could not be read (missing, only root can read it)
)

// Who wins when a user rule and a VPN route cover the same addresses.
const (
	WinnerRule   = "rule"   // the rule's prefix is longer: its route wins (longest prefix match)
	WinnerVPN    = "vpn"    // the VPN's prefix is longer: the VPN's route wins for that part
	WinnerSystem = "system" // same prefix on two interfaces: the OS picks by metric (no verificado per OS)
)

// RouteConflict is one active user rule whose IP range overlaps a VPN route
// while sending it through another profile.
type RouteConflict struct {
	Rule        string `json:"rule"`
	Via         string `json:"via"`
	Destination string `json:"destination"`
	Winner      string `json:"winner"`
}

// OwnRoute is one destination the VPN itself sends through its tunnel.
type OwnRoute struct {
	Destination string          `json:"destination"`
	Origin      string          `json:"origin"`
	Conflicts   []RouteConflict `json:"conflicts,omitempty"`
}

// ProfileOwnRoutes is what one profile brings by itself.
type ProfileOwnRoutes struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Connected bool   `json:"connected"`
	Source    string `json:"source"`
	// AllTraffic: the VPN takes the whole IPv4 default route (0.0.0.0/0 or
	// both /1 halves, redirect-gateway, or a consumer provider app).
	AllTraffic bool `json:"all_traffic"`
	// RulesOnly: a direct WireGuard profile without default_egress. Kongtrol
	// replaces the file's AllowedIPs with the tunnel subnet, its DNS and the
	// rule destinations (ProfileService.policyAllowedIPs), so the file's
	// routes are listed but not used.
	RulesOnly bool `json:"rules_only"`
	// ServerPending: an OpenVPN profile that is not connected; what the
	// server pushes is only known once it connects.
	ServerPending bool       `json:"server_pending"`
	Routes        []OwnRoute `json:"routes"`
}

// directTypes are the profile types whose routes Kongtrol can read.
var directTypes = map[string]bool{"openvpn": true, "wireguard": true}

// OwnRoutes lists, per profile (only one when only != ""), the routes the
// VPN brings by itself, with the user's active rules that overlap them.
// adapters may be nil (no daemon: nothing is connected).
func OwnRoutes(cfg *config.Config, adapters *vpn.Set, only string) []ProfileOwnRoutes {
	out := []ProfileOwnRoutes{}
	if cfg == nil {
		return out
	}
	names := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		if only == "" || name == only {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	rules := ruleRanges(cfg.ActivePolicies())
	for _, name := range names {
		v := cfg.VPNs[name]
		p := ProfileOwnRoutes{Name: name, Type: v.Type, Routes: []OwnRoute{}}
		var a vpn.VPNAdapter
		if adapters != nil {
			a, _ = adapters.Get(name)
		}
		if a != nil {
			p.Connected = a.Status().Normalize() == vpn.StatusConnected
		}
		if !directTypes[v.Type] {
			p.Source = OwnSourceApp
			p.AllTraffic = fullTunnelApps[v.Type]
			out = append(out, p)
			continue
		}
		p.Source = OwnSourceConfig
		var file []string
		var err error
		if v.Type == "openvpn" {
			file, err = openvpn.ConfigRoutes(v.ConfigFile)
		} else {
			file, err = wireguard.ConfigAllowedIPs(v.ConfigFile)
			if !v.DefaultEgress {
				if addr, aerr := wireguard.ParseConfigAddress(v.ConfigFile); aerr == nil && addr != nil {
					p.RulesOnly = true
				}
			}
		}
		if err != nil {
			p.Source = OwnSourceUnreadable
		}
		for _, c := range file {
			p.Routes = appendOwn(p.Routes, c, OwnRouteFile)
		}
		if v.Type == "openvpn" {
			p.ServerPending = !p.Connected
			if p.Connected {
				if rr, ok := a.(vpn.RouteReporter); ok {
					if tr := rr.TunnelRoutes(); tr != nil {
						if tr.AllTraffic {
							p.Routes = appendOwn(p.Routes, "0.0.0.0/0", OwnRouteServer)
						}
						for _, c := range tr.Routes {
							p.Routes = appendOwn(p.Routes, c, OwnRouteServer)
						}
					}
				}
			}
		}
		if v.Type == "wireguard" && v.DefaultEgress {
			p.AllTraffic = true
		}
		p.AllTraffic = p.AllTraffic || (!p.RulesOnly && CarriesAllTraffic(p.Routes))
		if !p.RulesOnly {
			for i := range p.Routes {
				p.Routes[i].Conflicts = routeConflicts(name, p.Routes[i].Destination, rules)
			}
		}
		out = append(out, p)
	}
	return out
}

func appendOwn(list []OwnRoute, cidr, origin string) []OwnRoute {
	pf, err := netip.ParsePrefix(cidr)
	if err != nil {
		return list
	}
	c := pf.Masked().String()
	if slices.ContainsFunc(list, func(r OwnRoute) bool { return r.Destination == c }) {
		return list
	}
	return append(list, OwnRoute{Destination: c, Origin: origin})
}

// CarriesAllTraffic reports whether routes take the whole IPv4 default
// route: 0.0.0.0/0, or both halves 0.0.0.0/1 + 128.0.0.0/1 (OpenVPN def1,
// Windows default_egress).
func CarriesAllTraffic(routes []OwnRoute) bool {
	half1, half2 := false, false
	for _, r := range routes {
		switch r.Destination {
		case "0.0.0.0/0":
			return true
		case "0.0.0.0/1":
			half1 = true
		case "128.0.0.0/1":
			half2 = true
		}
	}
	return half1 && half2
}

type ruleRange struct {
	rule, via string
	prefix    netip.Prefix
}

// ruleRanges are the IP ranges of the active rules. Domain rules are left
// out: their IPs are only known once resolved, and each resolved IP gets
// its own /32 route (monitor.PolicyResolver), which no VPN route beats.
func ruleRanges(policies []config.PolicyRule) []ruleRange {
	var out []ruleRange
	for _, p := range policies {
		for _, s := range p.Match.IPRanges {
			s = strings.TrimSpace(s)
			pf, err := netip.ParsePrefix(s)
			if err != nil {
				a, aerr := netip.ParseAddr(s)
				if aerr != nil {
					continue
				}
				pf = netip.PrefixFrom(a, a.BitLen())
			}
			out = append(out, ruleRange{rule: p.Name, via: p.Via, prefix: pf.Masked()})
		}
	}
	return out
}

// routeConflicts decides, for one VPN route of profile, which overlapping
// rule ranges (sent through another profile) win. The OS picks the longest
// prefix first; with equal prefixes on two interfaces it compares metrics
// (Windows: route + interface metric; Linux: route priority), which
// Kongtrol does not control on the VPN's side. A default route or half
// (/0, /1) is never in conflict: PolicyResolver never installs one for a
// rule, and every narrower rule route beats it. A rule through the same
// profile is not a conflict.
func routeConflicts(profile, dest string, rules []ruleRange) []RouteConflict {
	vp, err := netip.ParsePrefix(dest)
	if err != nil || vp.Bits() <= 1 {
		return nil
	}
	var out []RouteConflict
	for _, r := range rules {
		if r.via == profile || r.prefix.Bits() <= 1 || !r.prefix.Overlaps(vp) {
			continue
		}
		w := WinnerSystem
		switch {
		case r.prefix.Bits() > vp.Bits():
			w = WinnerRule
		case r.prefix.Bits() < vp.Bits():
			w = WinnerVPN
		}
		out = append(out, RouteConflict{Rule: r.rule, Via: r.via, Destination: r.prefix.String(), Winner: w})
	}
	return out
}
