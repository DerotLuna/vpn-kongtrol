package main

import "fmt"

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveProfiles returns the list of profile names from explicit args or a
// group, validated against the config before any work starts. verb is the
// command name ("up", "down") used in the "which VPN?" example.
func resolveProfiles(verb string, args []string, group string) ([]string, error) {
	if group != "" {
		if cfg == nil {
			return nil, fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
		}
		g, ok := cfg.Groups[group]
		if !ok {
			return nil, fmt.Errorf("%s", cf("cli.error.unknown_group", group))
		}
		if err := validateProfileNames(g.Profiles); err != nil {
			return nil, err
		}
		return g.Profiles, nil
	}
	if len(args) == 0 {
		return nil, specifyProfileError(verb)
	}
	if err := validateProfileNames(args); err != nil {
		return nil, err
	}
	return args, nil
}

func resolveUpProfiles(args []string, group string, useFavorites bool) ([]string, error) {
	if len(args) > 0 || group != "" {
		return resolveProfiles("up", args, group)
	}
	prefs, err := loadPreferences()
	if err == nil {
		if useFavorites && len(prefs.Favorites) > 0 {
			if err := validateProfileNames(prefs.Favorites); err != nil {
				return nil, err
			}
			return prefs.Favorites, nil
		}
		if prefs.DefaultGroup != "" {
			return resolveProfiles("up", nil, prefs.DefaultGroup)
		}
	}
	if useFavorites {
		return nil, fmt.Errorf("%s", ct("cli.error.no_favorites"))
	}
	return nil, specifyProfileError("up")
}
