package app

import (
	"net"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// KillSwitchService encapsulates the policy that decides when and how
// the kill switch should be enabled for active protected profiles.
type KillSwitchService struct {
	cfg      *config.Config
	adapters map[string]vpn.VPNAdapter
	ks       security.KillSwitch
}

func NewKillSwitchService(cfg *config.Config, adapters map[string]vpn.VPNAdapter, ks security.KillSwitch) *KillSwitchService {
	return &KillSwitchService{
		cfg:      cfg,
		adapters: adapters,
		ks:       ks,
	}
}

func (s *KillSwitchService) Apply() error {
	if s == nil || s.ks == nil || s.cfg == nil {
		return nil
	}

	var protectedConnected []string
	for name, a := range s.adapters {
		if !s.profileKillSwitchEnabled(name) {
			continue
		}
		if a.Status().Normalize() == vpn.StatusConnected {
			protectedConnected = append(protectedConnected, name)
		}
	}
	if len(protectedConnected) == 0 {
		return s.ks.Disable()
	}

	var tunnelIfaces []string
	var endpointIPs []string
	seenIface := map[string]bool{}
	seenEP := map[string]bool{}

	for _, name := range protectedConnected {
		if info, err := s.adapters[name].TunnelInfo(); err == nil && info != nil && info.InterfaceName != "" {
			if !seenIface[info.InterfaceName] {
				seenIface[info.InterfaceName] = true
				tunnelIfaces = append(tunnelIfaces, info.InterfaceName)
			}
		}
		for _, ep := range s.profileEndpointIPs(name) {
			if !seenEP[ep] {
				seenEP[ep] = true
				endpointIPs = append(endpointIPs, ep)
			}
		}
	}
	if len(tunnelIfaces) == 0 {
		return nil
	}

	allowSpec := strings.Join(tunnelIfaces, ",")
	if len(endpointIPs) > 0 {
		allowSpec += "|" + strings.Join(endpointIPs, ",")
	}
	return s.ks.Enable(allowSpec, s.cfg.Security.KillSwitch.AllowLAN)
}

func (s *KillSwitchService) profileKillSwitchEnabled(profile string) bool {
	v, ok := s.cfg.VPNs[profile]
	if !ok {
		return s.cfg.Security.KillSwitch.Enabled
	}
	if v.KillSwitch != nil {
		return *v.KillSwitch
	}
	return s.cfg.Security.KillSwitch.Enabled
}

func (s *KillSwitchService) profileEndpointIPs(name string) []string {
	vpnCfg, ok := s.cfg.VPNs[name]
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	add := func(ip string) {
		if ip != "" && !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}

	if vpnCfg.Host != "" {
		if ip := net.ParseIP(vpnCfg.Host); ip != nil {
			add(ip.String())
		} else if ips, err := net.LookupIP(vpnCfg.Host); err == nil {
			for _, ip := range ips {
				if ip4 := ip.To4(); ip4 != nil {
					add(ip4.String())
				}
			}
		}
	}
	if vpnCfg.Type == "wireguard" && vpnCfg.ConfigFile != "" {
		if epIP, err := wireguard.ParseEndpoint(vpnCfg.ConfigFile); err == nil && epIP != nil {
			add(epIP.String())
		}
	}
	return out
}
