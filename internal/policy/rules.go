// Package policy implements the traffic routing policy engine.
// It evaluates rules to determine which VPN adapter handles a given destination.
package policy

import (
	"fmt"
	"net"
)

// Rule maps a set of traffic patterns to a VPN profile name.
type Rule struct {
	Name     string
	Match    MatchSpec
	Via      string // VPN profile name from config
	Priority int    // higher = evaluated first
}

// MatchSpec defines which traffic a rule targets.
// A destination matches if it satisfies ANY ip_range OR ANY domain.
type MatchSpec struct {
	IPRanges []*net.IPNet
	Domains  []string // glob patterns: "*.example.com", "exact.host.com"
}

// ParseRule converts raw config strings into a compiled Rule.
func ParseRule(name, via string, ipRanges, domains []string, priority int) (*Rule, error) {
	r := &Rule{
		Name:     name,
		Via:      via,
		Priority: priority,
		Match: MatchSpec{
			Domains: domains,
		},
	}

	for _, cidr := range ipRanges {
		// Accept bare IPs (e.g. "172.28.152.26") by appending a host prefix.
		if net.ParseIP(cidr) != nil {
			if net.ParseIP(cidr).To4() != nil {
				cidr += "/32"
			} else {
				cidr += "/128"
			}
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: invalid CIDR %q: %w", name, cidr, err)
		}
		r.Match.IPRanges = append(r.Match.IPRanges, network)
	}

	return r, nil
}

// MatchesIP reports whether the rule matches the given IP address.
// Uses longest-prefix matching: the most specific (narrowest) subnet wins.
func (r *Rule) MatchesIP(ip net.IP) bool {
	for _, network := range r.Match.IPRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// MatchesDomain reports whether the rule matches the given domain name.
// Patterns are glob-style: "*.example.com" matches "sub.example.com".
func (r *Rule) MatchesDomain(domain string) bool {
	for _, pattern := range r.Match.Domains {
		if matchGlob(pattern, domain) {
			return true
		}
	}
	return false
}

// prefixLen returns the prefix length of a network for specificity comparison.
func prefixLen(n *net.IPNet) int {
	ones, _ := n.Mask.Size()
	return ones
}

// matchGlob performs simple glob matching where * matches any sequence of
// non-dot characters and ** matches any sequence including dots.
func matchGlob(pattern, s string) bool {
	// Exact match.
	if pattern == s {
		return true
	}
	// Wildcard prefix: *.example.com matches sub.example.com
	if len(pattern) > 2 && pattern[:2] == "*." {
		suffix := pattern[1:] // ".example.com"
		if len(s) > len(suffix) && s[len(s)-len(suffix):] == suffix {
			// Ensure the wildcard part has no dots (single label only).
			prefix := s[:len(s)-len(suffix)]
			for _, c := range prefix {
				if c == '.' {
					return false
				}
			}
			return true
		}
	}
	return false
}
