// Package policy implements the traffic routing policy engine.
// It evaluates rules to determine which VPN adapter handles a given destination.
package policy

import (
	"fmt"
	"net"
	"path"
	"strings"
)

// Rule maps a set of traffic patterns to a VPN profile name.
type Rule struct {
	Name        string
	Match       MatchSpec
	Via         string // VPN profile name from config
	Priority    int    // higher = evaluated first
	appPatterns []string
}

// MatchSpec defines which traffic a rule targets.
// A destination matches if it satisfies ANY ip_range OR ANY domain.
type MatchSpec struct {
	IPRanges []*net.IPNet
	Domains  []string // glob patterns: "*.example.com", "exact.host.com"
	Apps     []string // executable names or glob patterns: "chrome", "slack*", "*\\Code.exe"
}

// ParseRule converts raw config strings into a compiled Rule.
func ParseRule(name, via string, ipRanges, domains, apps []string, priority int) (*Rule, error) {
	r := &Rule{
		Name:     name,
		Via:      via,
		Priority: priority,
		Match: MatchSpec{
			Domains: domains,
			Apps:    apps,
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
	r.appPatterns = make([]string, len(apps))
	for i, app := range apps {
		r.appPatterns[i] = normalizePattern(app)
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

// MatchesApp reports whether the rule matches the given process executable.
// Accepts either full process path or executable name.
func (r *Rule) MatchesApp(app string) bool {
	full, base, baseNoExt := normalizeApp(app)
	for _, pattern := range r.appPatterns {
		if appMatchesNormalized(full, base, baseNoExt, pattern) {
			return true
		}
	}
	return false
}

// MatchesFlow reports whether the rule matches a combined flow context.
// If the rule defines app matchers, app must be provided and match.
// If the rule defines IP/domain matchers, target must be provided and match.
func (r *Rule) MatchesFlow(target string, app string) bool {
	needsApp := len(r.Match.Apps) > 0
	needsTarget := len(r.Match.IPRanges) > 0 || len(r.Match.Domains) > 0

	if needsApp {
		if strings.TrimSpace(app) == "" || !r.MatchesApp(app) {
			return false
		}
	}
	if needsTarget {
		target = strings.TrimSpace(target)
		if target == "" {
			return false
		}
		if ip := net.ParseIP(target); ip != nil {
			return r.MatchesIP(ip)
		}
		return r.MatchesDomain(target)
	}
	return needsApp
}

func appMatchesPattern(app, pattern string) bool {
	full, base, baseNoExt := normalizeApp(app)
	if full == "" {
		return false
	}
	p := normalizePattern(pattern)
	return appMatchesNormalized(full, base, baseNoExt, p)
}

func appMatchesNormalized(full, base, baseNoExt, pattern string) bool {
	if full == "" {
		return false
	}
	if pattern == "" {
		return false
	}
	if strings.Contains(pattern, "/") {
		if ok, _ := path.Match(pattern, full); ok {
			return true
		}
		return false
	}

	if ok, _ := path.Match(pattern, base); ok {
		return true
	}
	if baseNoExt != "" {
		if ok, _ := path.Match(pattern, baseNoExt); ok {
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

func normalizePattern(pattern string) string {
	p := strings.TrimSpace(pattern)
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.ToLower(p)
	return p
}

func normalizeApp(app string) (full, base, baseNoExt string) {
	full = normalizePattern(app)
	if full == "" {
		return "", "", ""
	}
	base = strings.ToLower(path.Base(full))
	if strings.HasSuffix(base, ".exe") {
		baseNoExt = strings.TrimSuffix(base, ".exe")
	}
	return full, base, baseNoExt
}
