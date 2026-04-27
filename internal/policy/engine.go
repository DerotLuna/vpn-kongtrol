package policy

import (
	"fmt"
	"net"
	"sort"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// Engine evaluates traffic policy rules and resolves a destination
// (IP or domain) to the VPN profile that should handle it.
type Engine struct {
	rules []*Rule // sorted by priority descending, then by prefix specificity
}

// New builds an Engine from the loaded configuration.
func New(cfg *config.Config) (*Engine, error) {
	e := &Engine{}

	for i, pr := range cfg.Policies {
		// Determine priority: explicit from config, or positional (earlier = higher).
		priority := len(cfg.Policies) - i
		if vpnCfg, ok := cfg.VPNs[pr.Via]; ok && vpnCfg.Priority > 0 {
			priority = vpnCfg.Priority
		}

		rule, err := ParseRule(pr.Name, pr.Via, pr.Match.IPRanges, pr.Match.Domains, priority)
		if err != nil {
			return nil, fmt.Errorf("policy: %w", err)
		}
		e.rules = append(e.rules, rule)
	}

	// Sort: higher priority first; among equal priority, more specific subnets first.
	sort.Slice(e.rules, func(i, j int) bool {
		if e.rules[i].Priority != e.rules[j].Priority {
			return e.rules[i].Priority > e.rules[j].Priority
		}
		// Tie-break by subnet specificity (longer prefix = more specific = first).
		maxI := maxPrefixLen(e.rules[i].Match.IPRanges)
		maxJ := maxPrefixLen(e.rules[j].Match.IPRanges)
		return maxI > maxJ
	})

	return e, nil
}

// ResolveIP returns the VPN profile name for a given destination IP.
// Returns ("", false) if no rule matches — traffic should go through
// the default physical interface.
func (e *Engine) ResolveIP(dst net.IP) (vpnName string, matched bool) {
	// Find the most specific matching rule (longest prefix).
	bestLen := -1
	bestVPN := ""

	for _, rule := range e.rules {
		for _, network := range rule.Match.IPRanges {
			if network.Contains(dst) {
				ones, _ := network.Mask.Size()
				if ones > bestLen {
					bestLen = ones
					bestVPN = rule.Via
				}
			}
		}
	}

	if bestVPN != "" {
		return bestVPN, true
	}
	return "", false
}

// ResolveDomain returns the VPN profile name for a given domain name.
// Returns ("", false) if no rule matches.
func (e *Engine) ResolveDomain(domain string) (vpnName string, matched bool) {
	for _, rule := range e.rules {
		if rule.MatchesDomain(domain) {
			return rule.Via, true
		}
	}
	return "", false
}

// Rules returns a copy of all rules loaded in the engine (for display purposes).
func (e *Engine) Rules() []Rule {
	out := make([]Rule, len(e.rules))
	for i, r := range e.rules {
		out[i] = *r // dereference: value copy, not pointer copy
	}
	return out
}

func maxPrefixLen(networks []*net.IPNet) int {
	max := 0
	for _, n := range networks {
		if l := prefixLen(n); l > max {
			max = l
		}
	}
	return max
}
