package privops

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits on request sizes. They bound work and memory per request.
const (
	MaxMetric        = 9999
	MaxDNSServers    = 8
	MaxKSInterfaces  = 16
	MaxKSEndpoints   = 64
	MaxAllowedIPs    = 1024
	MaxHostEntries   = 2048
	MaxBlockPrefixes = 4096
	MaxConfigSize    = 512 << 10 // .ovpn / .conf
	MaxReferenceSize = 256 << 10 // ca/cert/key files referenced by a .ovpn
	maxPathLen       = 4096
)

// invalid wraps ErrInvalidRequest with a reason.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
}

// ValidateInterfaceName checks a network interface name against a safe
// charset. Windows friendly names may contain letters in any script, digits,
// spaces and a few punctuation marks ("Ethernet 2", "Conexión de área local*").
// '=', quotes, slashes and control characters are rejected so a name can
// never be read as another netsh/route/pfctl argument.
func ValidateInterfaceName(name string) error {
	if name == "" || !utf8.ValidString(name) {
		return invalid("interface name is empty or not UTF-8")
	}
	if utf8.RuneCountInString(name) > maxIfaceRunes || len(name) > 4*maxIfaceRunes {
		return invalid("interface name too long")
	}
	if strings.TrimSpace(name) != name || strings.HasPrefix(name, "-") {
		return invalid("interface name has leading/trailing space or dash")
	}
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case strings.ContainsRune(" ._-()#+*", r):
			if r == ' ' && !ifaceAllowsSpace {
				return invalid("interface name %q has a space", name)
			}
		default:
			return invalid("interface name %q has a forbidden character %q", name, r)
		}
	}
	return nil
}

var wgNameRE = regexp.MustCompile(`^[a-zA-Z0-9_=+.-]{1,32}$`)

// ValidateWGName checks a WireGuard tunnel name (wireguard-windows rule;
// Linux additionally caps interface names at 15 bytes).
func ValidateWGName(name string) error {
	if !wgNameRE.MatchString(name) || strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") {
		return invalid("WireGuard tunnel name %q must be 1-32 of [A-Za-z0-9_=+.-]", name)
	}
	if len(name) > maxWGNameLen {
		return invalid("WireGuard tunnel name %q longer than %d", name, maxWGNameLen)
	}
	// `wg show all` / `wg show interfaces` are wg(8) keywords, not tunnels:
	// a tunnel with that name would make the helper read every tunnel.
	if strings.EqualFold(name, "all") || strings.EqualFold(name, "interfaces") {
		return invalid("WireGuard tunnel name %q is reserved by wg", name)
	}
	return nil
}

// ValidateWGKey checks a base64 Curve25519 public key.
func ValidateWGKey(k string) error {
	b, err := base64.StdEncoding.DecodeString(k)
	if err != nil || len(b) != 32 || len(k) != 44 {
		return invalid("peer public key is not a base64 32-byte key")
	}
	return nil
}

var sessionIDRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ValidateSessionID checks an OpenVPN session id.
func ValidateSessionID(id string) error {
	if !sessionIDRE.MatchString(id) {
		return invalid("bad session id")
	}
	return nil
}

func validUnicast(a netip.Addr) bool {
	return a.IsValid() && !a.IsUnspecified() && !a.IsMulticast() && !a.IsInterfaceLocalMulticast() &&
		!(a.Is4() && a == netip.AddrFrom4([4]byte{255, 255, 255, 255}))
}

// NormalizeRoute validates r and returns it with a masked prefix and
// unmapped addresses.
func NormalizeRoute(r Route) (Route, error) {
	if err := ValidateInterfaceName(r.Interface); err != nil {
		return r, err
	}
	if !r.Destination.IsValid() {
		return r, invalid("route destination is not a valid CIDR")
	}
	d := netip.PrefixFrom(r.Destination.Addr().Unmap(), r.Destination.Bits())
	if d.Addr().Is4In6() || !d.IsValid() {
		return r, invalid("route destination is not a valid CIDR")
	}
	d = d.Masked()
	if d.Addr().IsLoopback() || d.Addr().IsMulticast() || (d.Bits() > 0 && d.Addr().IsLinkLocalMulticast()) {
		return r, invalid("route destination %s is loopback or multicast", d)
	}
	r.Destination = d
	if r.Gateway.IsValid() {
		g := r.Gateway.Unmap()
		if !validUnicast(g) || g.IsLoopback() {
			return r, invalid("route gateway %s is not a unicast address", g)
		}
		if g.Is4() != d.Addr().Is4() {
			return r, invalid("route gateway and destination are different IP families")
		}
		r.Gateway = g
	}
	if r.Metric < 0 || r.Metric > MaxMetric {
		return r, invalid("route metric %d outside 0-%d", r.Metric, MaxMetric)
	}
	return r, nil
}

// NormalizeDNS validates req and deduplicates servers (order kept).
func NormalizeDNS(req DNSRequest) (DNSRequest, error) {
	if err := ValidateInterfaceName(req.Interface); err != nil {
		return req, err
	}
	if len(req.Servers) == 0 || len(req.Servers) > MaxDNSServers {
		return req, invalid("need 1-%d DNS servers", MaxDNSServers)
	}
	seen := map[netip.Addr]bool{}
	var out []netip.Addr
	for _, s := range req.Servers {
		s = s.Unmap()
		if !validUnicast(s) {
			return req, invalid("DNS server %s is not a unicast address", s)
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	req.Servers = out
	return req, nil
}

// NormalizeKillSwitch validates req.
func NormalizeKillSwitch(req KillSwitchRequest) (KillSwitchRequest, error) {
	if len(req.Interfaces) > MaxKSInterfaces || len(req.Endpoints) > MaxKSEndpoints {
		return req, invalid("too many kill switch interfaces or endpoints")
	}
	for _, i := range req.Interfaces {
		if err := ValidateInterfaceName(i); err != nil {
			return req, err
		}
	}
	for n, e := range req.Endpoints {
		e = e.Unmap()
		if !validUnicast(e) || e.IsLoopback() {
			return req, invalid("kill switch endpoint %s is not a unicast address", e)
		}
		req.Endpoints[n] = e
	}
	return req, nil
}

// NormalizeAllowedIPs validates req.
func NormalizeAllowedIPs(req WGAllowedIPsRequest) (WGAllowedIPsRequest, error) {
	if err := ValidateWGName(req.Interface); err != nil {
		return req, err
	}
	if err := ValidateWGKey(req.PeerPublicKey); err != nil {
		return req, err
	}
	if len(req.AllowedIPs) == 0 || len(req.AllowedIPs) > MaxAllowedIPs {
		return req, invalid("need 1-%d allowed IPs", MaxAllowedIPs)
	}
	for n, p := range req.AllowedIPs {
		if !p.IsValid() {
			return req, invalid("allowed IP is not a valid CIDR")
		}
		req.AllowedIPs[n] = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()).Masked()
		if !req.AllowedIPs[n].IsValid() {
			return req, invalid("allowed IP %s is not a valid CIDR", p)
		}
	}
	return req, nil
}

// ValidateConfigPath checks a user config path: absolute, clean, no NUL and
// one of the allowed extensions. Ownership is checked when the file is read.
func ValidateConfigPath(p string, exts ...string) error {
	if p == "" || len(p) > maxPathLen || strings.ContainsRune(p, 0) || !utf8.ValidString(p) {
		return invalid("config path is empty, too long or not UTF-8")
	}
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return invalid("config path %q must be absolute and clean", p)
	}
	if strings.HasPrefix(p, `\\`) {
		return invalid("config path %q must be a local file", p)
	}
	ext := strings.ToLower(filepath.Ext(p))
	for _, e := range exts {
		if ext == e {
			return nil
		}
	}
	return invalid("config path %q must end in %s", p, strings.Join(exts, " or "))
}

var hostLabelRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateHostname checks a lowercase DNS name for the hosts file.
func ValidateHostname(h string) error {
	if h == "" || len(h) > 253 {
		return invalid("host name is empty or too long")
	}
	for _, l := range strings.Split(h, ".") {
		if !hostLabelRE.MatchString(l) {
			return invalid("host name %q is not a valid lowercase DNS name", h)
		}
	}
	return nil
}

// NormalizeHostEntries validates entries and drops exact duplicates.
func NormalizeHostEntries(entries []HostEntry) ([]HostEntry, error) {
	if len(entries) > MaxHostEntries {
		return nil, invalid("more than %d host entries", MaxHostEntries)
	}
	seen := map[string]bool{}
	var out []HostEntry
	for _, e := range entries {
		e.IP = e.IP.Unmap()
		if !validUnicast(e.IP) {
			return nil, invalid("host entry IP %s is not a unicast address", e.IP)
		}
		if err := ValidateHostname(e.Host); err != nil {
			return nil, err
		}
		if seen[e.Host] {
			continue
		}
		seen[e.Host] = true
		out = append(out, e)
	}
	return out, nil
}
