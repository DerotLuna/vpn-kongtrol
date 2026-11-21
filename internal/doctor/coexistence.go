package doctor

import (
	"slices"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

var (
	activeProviderTunnel  = appattach.ActiveTunnel
	listNetworkInterfaces = appattach.Interfaces
	listSystemRoutes      = routing.ListAll
)

// Additional named interfaces for clients which do not use appattach's
// connection workflow. Generic tun/utun names cannot identify a provider.
var coexistenceMatchers = []struct {
	provider string
	matcher  appattach.Matcher
}{
	{"ciscoanyconnect", appattach.Matcher{Patterns: []string{"*Cisco AnyConnect*"}}},
	{"cloudflarewarp", appattach.Matcher{Patterns: []string{"*Cloudflare WARP*", "CloudflareWARP"}}},
	{"tailscale", appattach.Matcher{Patterns: []string{"Tailscale", "Tailscale Tunnel", "tailscale0"}}},
	{"tunnelbear", appattach.Matcher{Patterns: []string{"TunnelBear Adapter*", "TunnelBear Tunnel*"}}},
}

// checkVPNCoexistence inspects the OS only when diagnostics is requested.
// Installed clients alone are insufficient evidence of a routing conflict.
func (d *run) checkVPNCoexistence(cfg *config.Config) {
	active := make(map[string]string)
	if cfg != nil && d.opt.Daemon != nil {
		for name, iface := range d.opt.Daemon.TunnelInterfaces {
			if profile, ok := cfg.VPNs[name]; ok && iface != "" && d.opt.Daemon.Tunnels[name] == vpn.StatusConnected {
				active[iface] = profile.Type
			}
		}
	}
	for _, provider := range vpn.Registered() {
		if tunnel, up, _ := activeProviderTunnel(provider); up && tunnel.Name != "" {
			active[tunnel.Name] = provider
		}
	}
	ifaces := listNetworkInterfaces()
	for _, candidate := range coexistenceMatchers {
		for _, iface := range ifaces {
			if _, known := active[iface.Name]; known || !candidate.matcher.Match(iface) {
				continue
			}
			for _, addr := range iface.Addrs {
				if addr.IsGlobalUnicast() {
					active[iface.Name] = candidate.provider
					break
				}
			}
		}
	}
	if len(active) < 2 {
		return
	}

	label := d.t("doctor.coexistence.label")
	for _, provider := range active {
		if provider == "ciscoanyconnect" {
			d.warn(label, d.t("doctor.coexistence.cisco"), d.t("doctor.coexistence.hint"))
			break
		}
	}
	routes, err := listSystemRoutes()
	if err != nil {
		d.warn(label, d.t("doctor.coexistence.routes_unavailable"), d.t("doctor.hint.retry"))
		return
	}
	for _, conflict := range competingDefaultRoutes(active, routes) {
		d.warn(label, d.f("doctor.coexistence.default_routes", conflict.family, strings.Join(conflict.interfaces, ", ")),
			d.t("doctor.coexistence.hint"))
	}
}

type defaultConflict struct {
	family     string
	interfaces []string
}

// A /0 or two complementary /1s claim the whole address family. Require two
// recognized tunnel interfaces in the same family. Multiple physical routes
// and one split tunnel alongside one full tunnel are ordinary configurations.
func competingDefaultRoutes(active map[string]string, routes []routing.Route) []defaultConflict {
	const (
		lower = 1 << iota
		upper
		both = lower | upper
	)
	var conflicts []defaultConflict
	for _, family := range []struct {
		bits int
		name string
	}{{32, "IPv4"}, {128, "IPv6"}} {
		coverage := make(map[string]int)
		for _, route := range routes {
			if _, known := active[route.Interface]; !known {
				continue
			}
			prefix, bits := route.Destination.Mask.Size()
			if bits != family.bits {
				continue
			}
			switch prefix {
			case 0:
				coverage[route.Interface] |= both
			case 1:
				ip := route.Destination.IP
				if bits == 32 {
					ip = ip.To4()
				} else {
					ip = ip.To16()
				}
				if len(ip) > 0 {
					half := lower << (ip[0] >> 7)
					coverage[route.Interface] |= half
				}
			}
		}
		var interfaces []string
		for iface, mask := range coverage {
			if mask == both {
				interfaces = append(interfaces, active[iface]+" ("+iface+")")
			}
		}
		if len(interfaces) > 1 {
			slices.Sort(interfaces)
			conflicts = append(conflicts, defaultConflict{family.name, interfaces})
		}
	}
	return conflicts
}
