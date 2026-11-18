package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// maxNameRunes bounds names of profiles, policies, groups and rules.
const maxNameRunes = 64

// ValidateName checks a user-chosen name (VPN profile, policy, group,
// scheduler rule). Names are shown in the dashboards, the tray and the
// terminal and travel in URLs, so they may not hold control characters or
// the markup/quoting characters < > " ' ` & \ (defense in depth against
// injection in any client). Spaces and accents stay allowed.
func ValidateName(kind, name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > maxNameRunes || !utf8.ValidString(name) ||
		strings.ContainsAny(name, "<>\"'`&\\") {
		return newValidationError("config.err.name_chars", kind, name)
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return newValidationError("config.err.name_chars", kind, name)
		}
	}
	return nil
}

// ValidateAuditLogPath rejects a security.audit_log.path that is a folder
// (existing, or written with a trailing separator): it must name a file.
// Empty means the default.
func ValidateAuditLogPath(p string) error {
	if p == "" {
		return nil
	}
	home, _ := os.UserHomeDir()
	expanded := expandHome(p, home)
	if strings.HasSuffix(expanded, "/") || strings.HasSuffix(expanded, string(filepath.Separator)) {
		return newValidationError("config.err.audit_log_dir", p)
	}
	if fi, err := os.Stat(expanded); err == nil && fi.IsDir() {
		return newValidationError("config.err.audit_log_dir", p)
	}
	return nil
}

func validateSemantics(cfg *Config) error {
	if cfg.Monitor.Dashboard.Bind != "" {
		if err := ValidateDashboardBind(cfg.Monitor.Dashboard.Bind); err != nil {
			return err
		}
	}
	if err := ValidateAuditLogPath(cfg.Security.AuditLog.Path); err != nil {
		return err
	}
	for _, ep := range cfg.Security.PublicIPCheck.Endpoints {
		if err := ValidatePublicIPEndpoint(ep); err != nil {
			return err
		}
	}

	names := make([]string, 0, len(cfg.VPNs))
	for n := range cfg.VPNs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := ValidateName("vpn", n); err != nil {
			return err
		}
		if err := validateProvider(n, cfg.VPNs[n]); err != nil {
			return err
		}
	}
	for _, p := range cfg.Policies {
		if err := ValidateName("policy", p.Name); err != nil {
			return err
		}
	}
	for g := range cfg.Groups {
		if err := ValidateName("group", g); err != nil {
			return err
		}
	}
	for _, r := range cfg.Monitor.Scheduler.Rules {
		if r.Name != "" {
			if err := ValidateName("rule", r.Name); err != nil {
				return err
			}
		}
	}
	if err := validateFailover(cfg.VPNs, names); err != nil {
		return err
	}

	policyNames := make(map[string]struct{}, len(cfg.Policies))

	for i, pol := range cfg.Policies {
		if _, ok := cfg.VPNs[pol.Via]; !ok {
			return fmt.Errorf("policy %q references unknown profile %q", pol.Name, pol.Via)
		}
		if len(pol.Match.IPRanges) == 0 && len(pol.Match.Domains) == 0 && len(pol.Match.Apps) == 0 {
			return fmt.Errorf("policy %q must define at least one matcher (ip_ranges, domains, or apps)", pol.Name)
		}
		if err := ValidateOnTunnelDown(pol); err != nil {
			return err
		}
		if _, exists := policyNames[pol.Name]; exists {
			return fmt.Errorf("duplicate policy name %q at index %d", pol.Name, i)
		}
		policyNames[pol.Name] = struct{}{}
	}

	for groupName, group := range cfg.Groups {
		for _, profile := range group.Profiles {
			if _, ok := cfg.VPNs[profile]; !ok {
				return fmt.Errorf("group %q references unknown profile %q", groupName, profile)
			}
		}
	}

	allowedWeekdays := map[string]struct{}{
		"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {},
	}
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if rule.Name == "" {
			return fmt.Errorf("scheduler rule at index %d must define name", i)
		}
		if len(rule.Profiles) == 0 {
			return fmt.Errorf("scheduler rule %q must define at least one profile", rule.Name)
		}
		for _, p := range rule.Profiles {
			if _, ok := cfg.VPNs[p]; !ok {
				return fmt.Errorf("scheduler rule %q references unknown profile %q", rule.Name, p)
			}
		}
		for _, day := range rule.Weekdays {
			day = strings.ToLower(strings.TrimSpace(day))
			if _, ok := allowedWeekdays[day]; !ok {
				return fmt.Errorf("scheduler rule %q has invalid weekday %q", rule.Name, day)
			}
		}
		if rule.Start != "" {
			if err := parseHHMM(rule.Start); err != nil {
				return fmt.Errorf("scheduler rule %q has invalid start time %q", rule.Name, rule.Start)
			}
		}
		if rule.End != "" {
			if err := parseHHMM(rule.End); err != nil {
				return fmt.Errorf("scheduler rule %q has invalid end time %q", rule.Name, rule.End)
			}
		}
	}

	return nil
}

func parseHHMM(v string) error {
	if len(v) != 5 || v[2] != ':' {
		return fmt.Errorf("bad HH:MM")
	}
	h1, m1 := v[:2], v[3:]
	if h1 < "00" || h1 > "23" || m1 < "00" || m1 > "59" {
		return fmt.Errorf("bad HH:MM range")
	}
	return nil
}

// validateProvider checks the attach-mode fields of one profile: they only
// apply to provider types, and location selection only to providers whose
// official CLI can select one (nordvpn, protonvpn). Surfshark has no CLI
// and FortiClient connects to a configured gateway, so a location there
// would be silently ignored — reject it instead.
func validateProvider(name string, v VPNConfig) error {
	if !ProviderTypes[v.Type] {
		for _, f := range []struct {
			field string
			set   bool
		}{
			{"auto_connect", v.AutoConnect != nil}, {"country", v.Country != ""}, {"city", v.City != ""},
			{"group", v.Group != ""}, {"interface", v.Interface != ""}, {"wait_timeout", v.WaitTimeout != ""},
			{"app_path", v.AppPath != ""},
		} {
			if f.set {
				return newValidationError("config.err.provider_field", name, f.field)
			}
		}
		return nil
	}
	if v.WaitTimeout != "" {
		d, err := time.ParseDuration(v.WaitTimeout)
		if err != nil || d < 10*time.Second {
			return newValidationError("config.err.wait_timeout", name, v.WaitTimeout)
		}
	}
	if v.Group != "" && v.Type != "nordvpn" {
		return newValidationError("config.err.group_nord", name)
	}
	canSelect := v.Type == "nordvpn" || v.Type == "protonvpn"
	if !canSelect && (v.Country != "" || v.City != "" || v.Server != "") {
		return newValidationError("config.err.no_location", name, v.Type)
	}
	if v.Server != "" && (v.Country != "" || v.City != "" || v.Group != "") {
		return newValidationError("config.err.server_combo", name)
	}
	return nil
}

// ValidateOnTunnelDown checks a policy's on_tunnel_down (allow, block,
// connect; block needs destinations, not only apps). Exported so the
// dashboard API reports it translated before saving.
func ValidateOnTunnelDown(pol PolicyRule) error {
	switch pol.OnTunnelDown {
	case "", OnTunnelDownAllow, OnTunnelDownBlock, OnTunnelDownConnect:
	default:
		return newValidationError("config.err.on_tunnel_down", pol.Name, pol.OnTunnelDown)
	}
	if pol.OnTunnelDown == OnTunnelDownBlock && len(pol.Match.IPRanges) == 0 && len(pol.Match.Domains) == 0 {
		// Apps alone give no destination to block.
		return newValidationError("config.err.on_tunnel_down_apps", pol.Name)
	}
	return nil
}

// validateFailover checks every profile's failover list: each entry is an
// existing profile, not the profile itself, listed once, and the lists
// form no cycle (a → b → a). names is the sorted profile list, so the
// reported error is stable.
func validateFailover(vpns map[string]VPNConfig, names []string) error {
	for _, n := range names {
		seen := make(map[string]bool, len(vpns[n].Failover))
		for _, target := range vpns[n].Failover {
			switch {
			case target == n:
				return newValidationError("config.err.failover_self", n)
			case seen[target]:
				return newValidationError("config.err.failover_duplicate", n, target)
			}
			if _, ok := vpns[target]; !ok {
				return newValidationError("config.err.failover_unknown", n, target)
			}
			seen[target] = true
		}
	}
	// Depth-first search; state 1 = on the current path, 2 = done.
	state := make(map[string]int, len(names))
	var path []string
	var visit func(n string) []string
	visit = func(n string) []string {
		state[n] = 1
		path = append(path, n)
		for _, t := range vpns[n].Failover {
			switch state[t] {
			case 1:
				for i, p := range path {
					if p == t {
						return append(append([]string(nil), path[i:]...), t)
					}
				}
			case 0:
				if c := visit(t); c != nil {
					return c
				}
			}
		}
		path = path[:len(path)-1]
		state[n] = 2
		return nil
	}
	for _, n := range names {
		if state[n] == 0 {
			if c := visit(n); c != nil {
				return newValidationError("config.err.failover_cycle", c[0], strings.Join(c, " → "))
			}
		}
	}
	return nil
}
