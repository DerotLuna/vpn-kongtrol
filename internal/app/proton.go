package app

import (
	"io"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Same Proton account and shared tunnel addresses (ADR-0103, refines
// ADR-0099). A direct WireGuard/OpenVPN profile downloaded from a Proton
// account and Proton's own app use the same account and, for WireGuard,
// the same tunnel address (10.2.0.2): up together they cut each other's
// connections, split tunnel or not. Two direct profiles with one tunnel
// address cannot be up together either.

// profileFacts is what Kongtrol reads from a direct profile's file.
type profileFacts struct {
	provider string       // profileimport.ProviderForContent
	addrs    []netip.Addr // WireGuard [Interface] Address values
}

type factsEntry struct {
	mod   time.Time
	size  int64
	facts profileFacts
}

var (
	factsMu    sync.Mutex
	factsCache = map[string]factsEntry{}
)

// directProfileFacts reads v's file once per change (path + size + mtime):
// the daemon's handover check runs every few seconds.
func directProfileFacts(v config.VPNConfig) profileFacts {
	if (v.Type != "wireguard" && v.Type != "openvpn") || v.ConfigFile == "" {
		return profileFacts{}
	}
	fi, err := os.Stat(v.ConfigFile)
	if err != nil || !fi.Mode().IsRegular() {
		return profileFacts{}
	}
	key := v.Type + "\x00" + v.ConfigFile
	factsMu.Lock()
	e, ok := factsCache[key]
	factsMu.Unlock()
	if ok && e.size == fi.Size() && e.mod.Equal(fi.ModTime()) {
		return e.facts
	}
	f, err := os.Open(v.ConfigFile)
	if err != nil {
		return profileFacts{}
	}
	b, err := io.ReadAll(io.LimitReader(f, profileimport.MaxSize))
	_ = f.Close()
	if err != nil {
		return profileFacts{}
	}
	facts := profileFacts{provider: profileimport.ProviderForContent(v.Type, b)}
	if v.Type == "wireguard" {
		facts.addrs = profileimport.WireGuardAddresses(b)
	}
	factsMu.Lock()
	factsCache[key] = factsEntry{mod: fi.ModTime(), size: fi.Size(), facts: facts}
	factsMu.Unlock()
	return facts
}

// DirectProfileProvider names the provider whose account issued a direct
// profile's file ("protonvpn"), or "" (not a direct profile, unknown, or
// the file cannot be read). Computed from the file: no config field.
func DirectProfileProvider(v config.VPNConfig) string {
	return directProfileFacts(v).provider
}

// isDirectProton: a direct profile made from a Proton account's file.
func isDirectProton(v config.VPNConfig) bool {
	return DirectProfileProvider(v) == profileimport.ProviderProton
}

// protonAppTunnel returns Proton's app tunnel when it is up on an
// interface that is not one of Kongtrol's own tunnels (a direct .conf named
// "proton-ch" gets an interface Proton's matcher would also accept).
func protonAppTunnel(external []ExternalTunnel, managedIfaces []string) (ExternalTunnel, bool) {
	for _, e := range external {
		if e.Type != profileimport.ProviderProton {
			continue
		}
		if e.Interface != "" && slices.ContainsFunc(managedIfaces, func(i string) bool { return strings.EqualFold(i, e.Interface) }) {
			continue
		}
		return e, true
	}
	return ExternalTunnel{}, false
}

// connectedIfaces returns the interfaces of the connected profiles other
// than skip.
func connectedIfaces(adapters *vpn.Set, skip string) []string {
	var out []string
	for name, a := range adapters.Snapshot() {
		if name == skip || a.Status().Normalize() != vpn.StatusConnected {
			continue
		}
		if info, err := a.TunnelInfo(); err == nil && info != nil && info.InterfaceName != "" {
			out = append(out, info.InterfaceName)
		}
	}
	return out
}

// sharedAddress returns a connected direct profile (other than name) whose
// tunnel address is also one of v's, and that address.
func sharedAddress(cfg *config.Config, adapters *vpn.Set, name string, v config.VPNConfig) (string, netip.Addr) {
	mine := directProfileFacts(v).addrs
	if len(mine) == 0 || cfg == nil {
		return "", netip.Addr{}
	}
	snap := adapters.Snapshot()
	names := make([]string, 0, len(snap))
	for n := range snap {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, other := range names {
		if other == name || snap[other].Status().Normalize() != vpn.StatusConnected {
			continue
		}
		for _, a := range directProfileFacts(cfg.VPNs[other]).addrs {
			if slices.Contains(mine, a) {
				return other, a
			}
		}
	}
	return "", netip.Addr{}
}

func newProtonAppConflictError(profile string, app ExternalTunnel) *FullTunnelConflictError {
	return &FullTunnelConflictError{
		ValidationError: &config.ValidationError{Key: "api.err.connect_proton_app", Args: []any{profile}},
		Profile:         profile,
		Other:           TunnelClaim{Label: app.Label, Interface: app.Interface},
	}
}

func newSharedAddressError(profile, other string, addr netip.Addr) *FullTunnelConflictError {
	return &FullTunnelConflictError{
		ValidationError: &config.ValidationError{Key: "api.err.connect_same_address", Args: []any{profile, other, addr.String()}},
		Profile:         profile,
		Other:           TunnelClaim{Profile: other, Label: other},
	}
}

// ProtonHandover is a direct Proton profile Kongtrol disconnects because
// Proton's app connected (ADR-0103). AppProfile is a protonvpn profile of
// the config that now shows the app's tunnel ("" when there is none).
type ProtonHandover struct {
	Profile    string
	AppProfile string
}

// ProtonAppHandovers lists the connected direct Proton profiles while
// Proton's app tunnel is up on another interface. Cheap when none is
// connected: the provider app tunnels are listed only then.
func (s *ProfileService) ProtonAppHandovers() []ProtonHandover {
	if s == nil || s.cfg == nil || s.externalTunnels == nil {
		return nil
	}
	var cands []string
	for name, a := range s.adapters.Snapshot() {
		if a.Status().Normalize() == vpn.StatusConnected && isDirectProton(s.cfg.VPNs[name]) {
			cands = append(cands, name)
		}
	}
	if len(cands) == 0 {
		return nil
	}
	if _, up := protonAppTunnel(s.externalTunnels(), connectedIfaces(s.adapters, "")); !up {
		return nil
	}
	appProfile := ""
	var provs []string
	for name, v := range s.cfg.VPNs {
		if v.Type == profileimport.ProviderProton {
			provs = append(provs, name)
		}
	}
	if len(provs) > 0 {
		slices.Sort(provs)
		appProfile = provs[0]
	}
	slices.Sort(cands)
	out := make([]ProtonHandover, 0, len(cands))
	for _, c := range cands {
		out = append(out, ProtonHandover{Profile: c, AppProfile: appProfile})
	}
	return out
}

// ProtonAppMayBlock reports whether a split direct profile that stopped
// answering may be blocked by Proton's app: Proton's tunnel is up on
// another interface and the profile is not itself a Proton file. Proton's
// "advanced kill switch" blocks traffic outside its interface; whether it
// blocks another VPN adapter is not verified (ADR-0103), so this only
// drives a hint.
func (s *ProfileService) ProtonAppMayBlock(name string) bool {
	if s == nil || s.cfg == nil || s.externalTunnels == nil {
		return false
	}
	v, ok := s.cfg.VPNs[name]
	if !ok || (v.Type != "wireguard" && v.Type != "openvpn") || ProfileCarriesAllTraffic(v) || isDirectProton(v) {
		return false
	}
	_, up := protonAppTunnel(s.externalTunnels(), connectedIfaces(s.adapters, name))
	return up
}
