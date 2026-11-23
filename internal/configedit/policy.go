package configedit

import (
	"net"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// Policy errors are *config.ValidationError (key + args, a ".fix" line), so
// the dashboard and the CLI show them translated. The API maps
// ErrKeyPolicyExists to 409 and ErrKeyPolicyNotFound to 404.
const (
	ErrKeyPolicyExists   = "api.err.policy_exists"
	ErrKeyPolicyNotFound = "api.err.policy_not_found"
)

// PolicyPatch is the JSON body for policy create/update/test. config.PolicyRule
// carries only yaml tags, so decoding JSON straight into it silently dropped
// "ip_ranges"; pointer fields also let an update apply only what was sent.
type PolicyPatch struct {
	Name     *string `json:"name"`
	Via      *string `json:"via"`
	Disabled *bool   `json:"disabled"`
	// OnTunnelDown: empty (inherit) | allow | block | connect.
	OnTunnelDown *string `json:"on_tunnel_down"`
	Match        *struct {
		IPRanges *[]string `json:"ip_ranges"`
		Domains  *[]string `json:"domains"`
		Apps     *[]string `json:"apps"`
	} `json:"match"`
}

// ApplyTo overlays the fields present in p onto base and returns the result.
func (p PolicyPatch) ApplyTo(base config.PolicyRule) config.PolicyRule {
	if p.Name != nil {
		base.Name = *p.Name
	}
	if p.OnTunnelDown != nil {
		base.OnTunnelDown = *p.OnTunnelDown
	}
	if p.Disabled != nil {
		base.Disabled = *p.Disabled
	}
	if p.Via != nil {
		base.Via = *p.Via
	}
	if p.Match != nil {
		if p.Match.IPRanges != nil {
			base.Match.IPRanges = *p.Match.IPRanges
		}
		if p.Match.Domains != nil {
			base.Match.Domains = *p.Match.Domains
		}
		if p.Match.Apps != nil {
			base.Match.Apps = *p.Match.Apps
		}
	}
	return base
}

// NormalizePolicy trims every field and drops empty match entries.
func NormalizePolicy(r config.PolicyRule) config.PolicyRule {
	r.Name = strings.TrimSpace(r.Name)
	r.Via = strings.TrimSpace(r.Via)
	r.OnTunnelDown = strings.ToLower(strings.TrimSpace(r.OnTunnelDown))
	r.Match.Domains = nonEmpty(r.Match.Domains)
	r.Match.IPRanges = nonEmpty(r.Match.IPRanges)
	r.Match.Apps = nonEmpty(r.Match.Apps)
	return r
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// ClonePolicy deep-copies r so normalizing the copy never mutates the
// slices still referenced by the loaded config.
func ClonePolicy(r config.PolicyRule) config.PolicyRule {
	r.Match.IPRanges = append([]string(nil), r.Match.IPRanges...)
	r.Match.Domains = append([]string(nil), r.Match.Domains...)
	r.Match.Apps = append([]string(nil), r.Match.Apps...)
	return r
}

// ValidatePolicy checks one (normalized) policy against cfg's profiles.
func ValidatePolicy(rule config.PolicyRule, cfg *config.Config) error {
	if rule.Name == "" {
		return &config.ValidationError{Key: "api.err.name_required"}
	}
	if rule.Via == "" {
		return &config.ValidationError{Key: "api.err.policy_via_required"}
	}
	if _, ok := cfg.VPNs[rule.Via]; !ok {
		return &config.ValidationError{Key: "api.err.policy_via_unknown", Args: []any{rule.Via}}
	}
	if len(rule.Match.Domains) == 0 && len(rule.Match.IPRanges) == 0 && len(rule.Match.Apps) == 0 {
		return &config.ValidationError{Key: "api.err.policy_empty"}
	}
	for _, r := range rule.Match.IPRanges {
		if net.ParseIP(r) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(r); err != nil {
			return &config.ValidationError{Key: "api.err.policy_bad_range", Args: []any{r}}
		}
	}
	if err := config.ValidateOnTunnelDown(rule, cfg.VPNs[rule.Via]); err != nil {
		return err
	}
	_, err := policy.ParseRule(rule.Name, rule.Via, rule.Match.IPRanges, rule.Match.Domains, rule.Match.Apps, 1)
	return err
}

// FindPolicy returns the index of the policy named name (case-insensitive,
// like every policy name check), or -1.
func FindPolicy(cfg *config.Config, name string) int {
	for i := range cfg.Policies {
		if strings.EqualFold(cfg.Policies[i].Name, name) {
			return i
		}
	}
	return -1
}

// AddPolicy normalizes and validates rule and appends it to cfg.Policies
// (not saved: call Save). The name must be new.
func AddPolicy(cfg *config.Config, rule config.PolicyRule) (config.PolicyRule, error) {
	rule = NormalizePolicy(ClonePolicy(rule))
	if err := ValidatePolicy(rule, cfg); err != nil {
		return rule, err
	}
	if FindPolicy(cfg, rule.Name) >= 0 {
		return rule, &config.ValidationError{Key: ErrKeyPolicyExists, Args: []any{rule.Name}}
	}
	cfg.Policies = append(cfg.Policies, rule)
	return rule, nil
}

// RemovePolicy drops the policy named name from cfg.Policies (not saved:
// call Save).
func RemovePolicy(cfg *config.Config, name string) error {
	i := FindPolicy(cfg, name)
	if i < 0 {
		return &config.ValidationError{Key: ErrKeyPolicyNotFound, Args: []any{name}}
	}
	cfg.Policies = append(cfg.Policies[:i:i], cfg.Policies[i+1:]...)
	return nil
}

// ErrKeyPolicyMoveSame refuses moving a profile's rules to itself.
const ErrKeyPolicyMoveSame = "api.err.policy_move_same"

// MovePolicies points every rule that goes through from to to instead
// (ADR-0104: a direct profile's rules move to the provider-app profile of
// the same account). Both profiles must exist; each moved rule is checked
// again against to. Returns the moved rule names, in config order; none is
// not an error. Not saved: call Save.
func MovePolicies(cfg *config.Config, from, to string) ([]string, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	for _, name := range []string{from, to} {
		if name == "" {
			return nil, &config.ValidationError{Key: "api.err.policy_via_required"}
		}
		if _, ok := cfg.VPNs[name]; !ok {
			return nil, &config.ValidationError{Key: "api.err.policy_via_unknown", Args: []any{name}}
		}
	}
	if from == to {
		return nil, &config.ValidationError{Key: ErrKeyPolicyMoveSame, Args: []any{from}}
	}
	moved := []string{}
	next := make([]config.PolicyRule, len(cfg.Policies))
	copy(next, cfg.Policies)
	for i := range next {
		if next[i].Via != from {
			continue
		}
		rule := ClonePolicy(next[i])
		rule.Via = to
		if err := ValidatePolicy(NormalizePolicy(rule), cfg); err != nil {
			return nil, err
		}
		next[i] = rule
		moved = append(moved, rule.Name)
	}
	cfg.Policies = next
	return moved, nil
}

// CountPolicies is how many rules go through profile name.
func CountPolicies(cfg *config.Config, name string) int {
	n := 0
	for _, p := range cfg.Policies {
		if p.Via == name {
			n++
		}
	}
	return n
}
