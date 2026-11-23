package profileimport

import (
	"net/netip"
	"strings"
)

// ProviderProton is the adapter type of Proton VPN, returned by
// ProviderForContent for a file downloaded from a Proton account.
const ProviderProton = "protonvpn"

// protonConfKeys are the comments Proton writes under [Interface] in every
// WireGuard file it generates ("# NetShield = 1", …). Proton does not
// document the format, so it may change (docs/research/2026-11-23-proton-same-account.md).
var protonConfKeys = []string{"bouncing", "netshield", "moderate nat", "nat-pmp", "vpn accelerator"}

// ProviderForContent names the VPN provider whose account issued a profile
// file, or "" when nothing marks it (ADR-0103). Only Proton VPN is known:
//   - WireGuard: two or more of Proton's comment keys (NetShield, VPN
//     Accelerator, Moderate NAT, NAT-PMP, Bouncing) and an address
//     10.x.0.2 or a DNS 10.x.0.1 (Proton's tunnel addressing);
//   - OpenVPN: a host ending in .protonvpn.net in `verify-x509-name` or
//     `remote`.
//
// typ is "wireguard" or "openvpn" (TypeForContent). It is a hint for the
// user and for coexistence checks, never a security decision.
func ProviderForContent(typ string, content []byte) string {
	switch typ {
	case "wireguard":
		if protonWireGuard(content) {
			return ProviderProton
		}
	case "openvpn":
		if protonOpenVPN(content) {
			return ProviderProton
		}
	}
	return ""
}

// HintKey is the i18n key of the line shown after importing a file from
// provider's account (Result.Provider), "" when there is nothing to say.
func HintKey(provider string) string {
	if provider == ProviderProton {
		return "profile.import.proton_hint"
	}
	return ""
}

func protonWireGuard(content []byte) bool {
	keys := map[string]bool{}
	addr := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if c, ok := strings.CutPrefix(line, "#"); ok {
			k, _, found := strings.Cut(c, "=")
			if !found {
				continue
			}
			k = strings.ToLower(strings.TrimSpace(k))
			for _, pk := range protonConfKeys {
				if strings.HasPrefix(k, pk) {
					keys[pk] = true
				}
			}
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "address":
			for _, a := range splitList(v) {
				if p, err := netip.ParsePrefix(a); err == nil && protonAddr(p.Addr(), 2) {
					addr = true
				} else if ip, err := netip.ParseAddr(a); err == nil && protonAddr(ip, 2) {
					addr = true
				}
			}
		case "dns":
			for _, a := range splitList(v) {
				if ip, err := netip.ParseAddr(a); err == nil && protonAddr(ip, 1) {
					addr = true
				}
			}
		}
	}
	return len(keys) >= 2 && addr
}

// protonAddr: 10.x.0.<last> (10.2.0.2 / 10.2.0.1, or 10.3.0.x for a second
// tunnel as Proton suggests).
func protonAddr(ip netip.Addr, last byte) bool {
	if !ip.Is4() {
		return false
	}
	b := ip.As4()
	return b[0] == 10 && b[2] == 0 && b[3] == last
}

func protonOpenVPN(content []byte) bool {
	for _, raw := range strings.Split(string(content), "\n") {
		f := strings.Fields(strings.TrimSpace(raw))
		if len(f) < 2 {
			continue
		}
		switch strings.ToLower(strings.TrimPrefix(f[0], "--")) {
		case "remote", "verify-x509-name":
			host := strings.ToLower(strings.TrimSuffix(strings.Trim(f[1], `"'`), "."))
			if strings.HasSuffix(host, ".protonvpn.net") {
				return true
			}
		}
	}
	return false
}

// WireGuardAddresses returns the tunnel addresses of a WireGuard file's
// [Interface] (its Address lines, several values or lines allowed), so two
// direct profiles with the same address are never up together (ADR-0103).
func WireGuardAddresses(content []byte) []netip.Addr {
	var out []netip.Addr
	inInterface := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			inInterface = strings.EqualFold(line, "[Interface]")
			continue
		}
		if !inInterface {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(k), "address") {
			continue
		}
		for _, a := range splitList(v) {
			if p, err := netip.ParsePrefix(a); err == nil {
				out = append(out, p.Addr().Unmap())
			} else if ip, err := netip.ParseAddr(a); err == nil {
				out = append(out, ip.Unmap())
			}
		}
	}
	return out
}

// splitList splits "a, b,c" (a value with an optional trailing comment).
func splitList(v string) []string {
	if i := strings.Index(v, "#"); i >= 0 {
		v = v[:i]
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
