package policy

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// Engine evaluates traffic policy rules and resolves a destination
// (IP or domain) to the VPN profile that should handle it.
type Engine struct {
	rules []*Rule // sorted by priority descending, then by prefix specificity
}

// ExplainResult contains human-friendly resolution metadata for a target.
type ExplainResult struct {
	Target    string
	Kind      string // ip | domain | app
	Matched   bool
	Via       string
	RuleName  string
	Reason    string
	DefaultTo string
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

		rule, err := ParseRule(pr.Name, pr.Via, pr.Match.IPRanges, pr.Match.Domains, pr.Match.Apps, priority)
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
	rule, _, ok := e.resolveIPMatch(dst)
	if ok {
		return rule.Via, true
	}
	return "", false
}

// ResolveDomain returns the VPN profile name for a given domain name.
// Returns ("", false) if no rule matches.
func (e *Engine) ResolveDomain(domain string) (vpnName string, matched bool) {
	rule, _, ok := e.resolveDomainMatch(domain)
	if ok {
		return rule.Via, true
	}
	return "", false
}

// ResolveApp returns the VPN profile name for a given process executable
// (name or full path). Returns ("", false) if no app rule matches.
func (e *Engine) ResolveApp(app string) (vpnName string, matched bool) {
	rule, _, ok := e.resolveAppMatch(app)
	if ok {
		return rule.Via, true
	}
	return "", false
}

// ResolveFlow returns the VPN profile and rule for a combined flow context.
// target may be an IP or domain; app may be an executable name/path.
func (e *Engine) ResolveFlow(target, app string) (vpnName string, ruleName string, matched bool) {
	for _, r := range e.rules {
		if r.MatchesFlow(target, app) {
			return r.Via, r.Name, true
		}
	}
	return "", "", false
}

// ExplainTarget explains how a target is resolved.
// Targets are interpreted in this order: IP, app:<exe>, domain.
func (e *Engine) ExplainTarget(target string) ExplainResult {
	target = strings.TrimSpace(target)
	if target == "" {
		return ExplainResult{Target: target, Reason: "empty target", DefaultTo: "default route"}
	}

	if ip := net.ParseIP(target); ip != nil {
		return e.ExplainIP(ip)
	}

	if strings.HasPrefix(strings.ToLower(target), "app:") {
		app := strings.TrimSpace(target[4:])
		return e.ExplainApp(app)
	}

	return e.ExplainDomain(target)
}

// ExplainIP explains IP resolution and includes longest-prefix detail.
func (e *Engine) ExplainIP(ip net.IP) ExplainResult {
	out := ExplainResult{
		Target:    ip.String(),
		Kind:      "ip",
		DefaultTo: "default route",
	}
	rule, network, ok := e.resolveIPMatch(ip)
	if !ok {
		out.Reason = "no matching ip_ranges rule"
		return out
	}
	out.Matched = true
	out.Via = rule.Via
	out.RuleName = rule.Name
	out.Reason = fmt.Sprintf("matched CIDR %s (longest-prefix wins)", network.String())
	return out
}

// ExplainDomain explains domain resolution and includes pattern detail.
func (e *Engine) ExplainDomain(domain string) ExplainResult {
	out := ExplainResult{
		Target:    domain,
		Kind:      "domain",
		DefaultTo: "default route",
	}
	rule, pattern, ok := e.resolveDomainMatch(domain)
	if !ok {
		out.Reason = "no matching domains rule"
		return out
	}
	out.Matched = true
	out.Via = rule.Via
	out.RuleName = rule.Name
	out.Reason = fmt.Sprintf("matched domain pattern %q", pattern)
	return out
}

// ExplainApp explains app resolution and includes pattern detail.
func (e *Engine) ExplainApp(app string) ExplainResult {
	out := ExplainResult{
		Target:    app,
		Kind:      "app",
		DefaultTo: "default route",
	}
	rule, pattern, ok := e.resolveAppMatch(app)
	if !ok {
		out.Reason = "no matching apps rule"
		return out
	}
	out.Matched = true
	out.Via = rule.Via
	out.RuleName = rule.Name
	out.Reason = fmt.Sprintf("matched app pattern %q", pattern)
	return out
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

func (e *Engine) resolveIPMatch(dst net.IP) (rule *Rule, network *net.IPNet, matched bool) {
	bestLen := -1
	var bestRule *Rule
	var bestNet *net.IPNet

	for _, r := range e.rules {
		for _, n := range r.Match.IPRanges {
			if !n.Contains(dst) {
				continue
			}
			ones, _ := n.Mask.Size()
			if ones > bestLen {
				bestLen = ones
				bestRule = r
				bestNet = n
			}
		}
	}

	if bestRule == nil {
		return nil, nil, false
	}
	return bestRule, bestNet, true
}

func (e *Engine) resolveDomainMatch(domain string) (rule *Rule, pattern string, matched bool) {
	for _, r := range e.rules {
		for _, p := range r.Match.Domains {
			if matchGlob(p, domain) {
				return r, p, true
			}
		}
	}
	return nil, "", false
}

func (e *Engine) resolveAppMatch(app string) (rule *Rule, pattern string, matched bool) {
	full, base, baseNoExt := normalizeApp(app)
	for _, r := range e.rules {
		for i, p := range r.appPatterns {
			if appMatchesNormalized(full, base, baseNoExt, p) {
				return r, r.Match.Apps[i], true
			}
		}
	}
	return nil, "", false
}
