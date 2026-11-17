package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func validateSemantics(cfg *Config) error {
	if cfg.Monitor.Dashboard.Bind != "" {
		if err := ValidateDashboardBind(cfg.Monitor.Dashboard.Bind); err != nil {
			return err
		}
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
		if err := validateProvider(n, cfg.VPNs[n]); err != nil {
			return err
		}
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
