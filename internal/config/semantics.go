package config

import (
	"fmt"
	"net/netip"
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
	if n := cfg.Security.AuditLog.MaxSizeMB; n < 0 || n > MaxAuditMaxSizeMB {
		return newValidationError("config.err.audit_max_size", n)
	}
	// 0 = the default (older configs keep loading, ADR-0064).
	if n := cfg.Security.AuditLog.RetentionDays; n < 0 || n > MaxAuditRetentionDays {
		return newValidationError("config.err.audit_retention", n)
	}
	if n := cfg.Security.AuditLog.MaxTotalMB; n < 0 || n > MaxAuditMaxTotalMB {
		return newValidationError("config.err.audit_max_total", n)
	}
	if al := cfg.Security.AuditLog; al.MaxTotalMB > 0 && al.MaxTotalMB < al.EffectiveMaxSizeMB() {
		return newValidationError("config.err.audit_total_small", al.MaxTotalMB, al.EffectiveMaxSizeMB())
	}
	if err := validateExpectedIPs(cfg); err != nil {
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
	if err := validateDefaultEgress(cfg.VPNs, names); err != nil {
		return err
	}
	for _, n := range names {
		if err := ValidateName("vpn", n); err != nil {
			return err
		}
		if err := validateProvider(n, cfg.VPNs[n]); err != nil {
			return err
		}
		if action := cfg.VPNs[n].OnTunnelDown; !validTunnelDownAction(action) {
			return newValidationError("config.err.profile_on_tunnel_down", n, action)
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

	for _, pol := range cfg.Policies {
		if _, ok := cfg.VPNs[pol.Via]; !ok {
			return newValidationError("config.err.policy_unknown_profile", pol.Name, pol.Via)
		}
		if len(pol.Match.IPRanges) == 0 && len(pol.Match.Domains) == 0 && len(pol.Match.Apps) == 0 {
			return newValidationError("config.err.policy_no_match", pol.Name)
		}
		if err := ValidateOnTunnelDown(pol, cfg.VPNs[pol.Via]); err != nil {
			return err
		}
		if _, exists := policyNames[pol.Name]; exists {
			return newValidationError("config.err.policy_duplicate", pol.Name)
		}
		policyNames[pol.Name] = struct{}{}
	}

	groupNames := make([]string, 0, len(cfg.Groups))
	for g := range cfg.Groups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames) // the same config always reports the same error
	for _, groupName := range groupNames {
		group := cfg.Groups[groupName]
		if len(group.Profiles) == 0 {
			return newValidationError("config.err.group_empty", groupName)
		}
		for _, profile := range group.Profiles {
			if _, ok := cfg.VPNs[profile]; !ok {
				return newValidationError("config.err.group_unknown_profile", groupName, profile)
			}
		}
	}

	allowedWeekdays := map[string]struct{}{
		"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {},
	}
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if rule.Name == "" {
			return newValidationError("config.err.rule_no_name", i+1)
		}
		if len(rule.Profiles) == 0 {
			return newValidationError("config.err.rule_no_profiles", rule.Name)
		}
		for _, p := range rule.Profiles {
			if _, ok := cfg.VPNs[p]; !ok {
				return newValidationError("config.err.rule_unknown_profile", rule.Name, p)
			}
		}
		for _, day := range rule.Weekdays {
			day = strings.ToLower(strings.TrimSpace(day))
			if _, ok := allowedWeekdays[day]; !ok {
				return newValidationError("config.err.rule_weekday", rule.Name, day)
			}
		}
		if rule.Start != "" {
			if err := parseHHMM(rule.Start); err != nil {
				return newValidationError("config.err.rule_time", rule.Name, rule.Start)
			}
		}
		if rule.End != "" {
			if err := parseHHMM(rule.End); err != nil {
				return newValidationError("config.err.rule_time", rule.Name, rule.End)
			}
		}
	}

	return nil
}

// validateExpectedIPs checks security.integrity_check.expected_ips: every
// key is an existing profile and every value one IP address (the leak
// check compares the public IP with it).
func validateExpectedIPs(cfg *Config) error {
	names := make([]string, 0, len(cfg.Security.IntegrityCheck.ExpectedIPs))
	for n := range cfg.Security.IntegrityCheck.ExpectedIPs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if _, ok := cfg.VPNs[n]; !ok {
			return newValidationError("config.err.expected_ip_profile", n)
		}
		ip := cfg.Security.IntegrityCheck.ExpectedIPs[n]
		if _, err := netip.ParseAddr(strings.TrimSpace(ip)); err != nil {
			return newValidationError("config.err.expected_ip", n, ip)
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

func validTunnelDownAction(action string) bool {
	return action == "" || action == OnTunnelDownAllow || BlocksWhenDown(action)
}

// ValidateOnTunnelDown validates the explicit action and the requirements of
// the effective action, including when inherited from the profile.
func ValidateOnTunnelDown(pol PolicyRule, profile VPNConfig) error {
	if !validTunnelDownAction(pol.OnTunnelDown) {
		return newValidationError("config.err.on_tunnel_down", pol.Name, pol.OnTunnelDown)
	}
	if BlocksWhenDown(pol.TunnelDownAction(profile)) && len(pol.Match.IPRanges) == 0 && len(pol.Match.Domains) == 0 {
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
