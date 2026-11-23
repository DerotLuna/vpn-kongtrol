package app

import (
	"sort"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// RuleMoveCandidate is a direct profile whose file the provider issued and
// that has rules (ADR-0104).
type RuleMoveCandidate struct {
	Name  string
	Rules int
}

// RuleMoveCandidates lists, by name, the direct profiles made from
// providerType's files (DirectProfileProvider) that have at least one rule:
// after a provider-app profile is added, Kongtrol offers to move their
// rules to it. The panel computes the same from GET /api/v1/vpns
// "provider" and GET /api/v1/policies.
func RuleMoveCandidates(cfg *config.Config, providerType string) []RuleMoveCandidate {
	if cfg == nil || providerType == "" {
		return nil
	}
	count := map[string]int{}
	for _, p := range cfg.Policies {
		count[p.Via]++
	}
	var out []RuleMoveCandidate
	for name, v := range cfg.VPNs {
		if count[name] == 0 || DirectProfileProvider(v) != providerType {
			continue
		}
		out = append(out, RuleMoveCandidate{Name: name, Rules: count[name]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
