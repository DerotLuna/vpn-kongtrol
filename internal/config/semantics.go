package config

import (
	"fmt"
	"strings"
)

func validateSemantics(cfg *Config) error {
	if cfg.Monitor.Dashboard.Bind != "" {
		if err := ValidateDashboardBind(cfg.Monitor.Dashboard.Bind); err != nil {
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
