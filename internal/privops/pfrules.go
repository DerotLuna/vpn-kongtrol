package privops

// Pure pf (macOS packet filter) text generation for the kill switch. It has
// no build tags so its tests run on every OS; only sys_darwin.go feeds the
// result to pfctl.

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// pfAnchor holds the kill switch rules. The reverse-DNS name keeps it
	// apart from Apple's own com.apple/* anchors.
	pfAnchor = "com.kongtrol"
	// pfAnchorLine is the one filter rule added to the system ruleset.
	pfAnchorLine    = `anchor "` + pfAnchor + `"`
	pfAnchorComment = "# Kongtrol kill switch (removed when it is turned off)"
	// maxPFConf caps how much of /etc/pf.conf is read.
	maxPFConf = 1 << 20
)

// pfMainRuleset returns the system ruleset base (normally /etc/pf.conf,
// including Apple's anchors) with `anchor "com.kongtrol"` appended. Every
// original line is kept as is; earlier copies of the Kongtrol lines are
// dropped, so the anchor appears exactly once however often this runs.
func pfMainRuleset(base []byte) []byte {
	var b strings.Builder
	for _, line := range strings.SplitAfter(string(base), "\n") {
		if t := strings.TrimSpace(line); t == pfAnchorLine || t == pfAnchorComment {
			continue
		}
		b.WriteString(line)
	}
	if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(pfAnchorComment + "\n" + pfAnchorLine + "\n")
	return []byte(b.String())
}

// pfIfaceRE accepts BSD interface names only (driver name + unit number:
// lo0, en0, utun4, bridge100). It is stricter than ValidateInterfaceName on
// purpose: pf would read '#' as a comment and a group name such as
// "egress" as every physical interface.
var pfIfaceRE = regexp.MustCompile(`^[a-z]{1,12}[0-9]{1,4}$`)

// pfKillSwitchRules returns the anchor rules for req. Pass rules come first
// and every rule is "quick", so the final block only catches what no pass
// rule matched. It covers IPv4 and IPv6: a tunnel without IPv6 routes
// cannot leak IPv6 through the physical interface.
func pfKillSwitchRules(req KillSwitchRequest) (string, error) {
	var b strings.Builder
	b.WriteString("# Kongtrol kill switch\n")
	b.WriteString("pass out quick on lo0 all\n")
	for _, i := range req.Interfaces {
		if !pfIfaceRE.MatchString(i) {
			return "", invalid("interface name %q is not valid for pf", i)
		}
		fmt.Fprintf(&b, "pass out quick on %s all\n", i)
	}
	for _, e := range req.Endpoints {
		e = e.Unmap()
		if !e.IsValid() || e.Zone() != "" || e.IsLoopback() || e.IsUnspecified() {
			return "", invalid("kill switch endpoint %s is not valid for pf", e)
		}
		fam := "inet"
		if e.Is6() {
			fam = "inet6"
		}
		fmt.Fprintf(&b, "pass out quick %s proto { udp tcp } to %s\n", fam, e)
	}
	// DHCP/DHCPv6 lease renewal and IPv6 neighbour discovery keep the
	// physical link (and so the tunnel) alive; they carry no user traffic.
	b.WriteString("pass out quick inet proto udp from any port 68 to any port 67\n")
	b.WriteString("pass out quick inet6 proto udp from any port 546 to any port 547\n")
	b.WriteString("pass out quick inet6 proto icmp6 all icmp6-type { routersol neighbrsol neighbradv }\n")
	if req.AllowLAN {
		// Private and link-local ranges plus link-scoped multicast and
		// broadcast, none of which can be routed off the local network.
		b.WriteString("pass out quick inet to { 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 169.254.0.0/16 224.0.0.0/24 255.255.255.255 }\n")
		b.WriteString("pass out quick inet6 to { fe80::/10 fc00::/7 ff02::/16 }\n")
	}
	b.WriteString("block drop out quick all\n")
	return b.String(), nil
}

// pfTokenRE matches the reference token `pfctl -E` prints ("Token : 123").
var pfTokenRE = regexp.MustCompile(`(?m)^\s*Token\s*:\s*([0-9]{1,20})\s*$`)

// parsePFToken extracts the enable token from `pfctl -E` output. The token
// is later given to `pfctl -X` so pf stays enabled for anyone else using it.
func parsePFToken(out string) (string, error) {
	m := pfTokenRE.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("%w: pfctl -E printed no token: %s", ErrFailed, strings.TrimSpace(out))
	}
	return m[1], nil
}
