package main

import "fmt"

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveProfiles returns the list of profile names from explicit args or a group.
func resolveProfiles(args []string, group string) ([]string, error) {
	if group != "" {
		if cfg == nil {
			return nil, fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
		}
		g, ok := cfg.Groups[group]
		if !ok {
			return nil, fmt.Errorf("%s", cf("cli.error.unknown_group", group))
		}
		return g.Profiles, nil
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", ct("cli.error.specify_profile_or_group"))
	}
	return args, nil
}

func resolveUpProfiles(args []string, group string, useFavorites bool) ([]string, error) {
	if len(args) > 0 || group != "" {
		return resolveProfiles(args, group)
	}
	prefs, err := loadPreferences()
	if err == nil {
		if useFavorites && len(prefs.Favorites) > 0 {
			return prefs.Favorites, nil
		}
		if prefs.DefaultGroup != "" {
			return resolveProfiles(nil, prefs.DefaultGroup)
		}
	}
	if useFavorites {
		return nil, fmt.Errorf("%s", ct("cli.error.no_favorites"))
	}
	return nil, fmt.Errorf("%s", ct("cli.error.specify_profile_or_group"))
}
