package main

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestConditionalHelperInstallAction(t *testing.T) {
	cases := []struct {
		name          string
		res           privops.PingResult
		err           error
		ifOutdated    helperInstallMode
		ifOutdatedKey string
		ifNeeded      helperInstallMode
		ifNeededKey   string
	}{
		{"not installed or stopped", privops.PingResult{}, privops.ErrHelperNotInstalled,
			helperModeCurrent, "service.install.nothing_to_update", helperModeInstall, ""},
		{"current protocol, other build", privops.PingResult{Protocol: privops.ProtocolVersion, Version: "other"}, nil,
			helperModeCurrent, "service.install.up_to_date", helperModeCurrent, "service.install.up_to_date"},
		{"older protocol", privops.PingResult{Protocol: privops.ProtocolVersion - 1}, nil,
			helperModeUpdate, "", helperModeUpdate, ""},
		{"client reports outdated", privops.PingResult{}, privops.ErrHelperOutdated,
			helperModeUpdate, "", helperModeUpdate, ""},
		{"denied", privops.PingResult{}, privops.ErrDenied,
			helperModeCurrent, "service.install.nothing_to_update", helperModeCurrent, "service.install.nothing_to_update"},
	}
	for _, c := range cases {
		mode, key := conditionalHelperInstallAction(c.res, c.err, false)
		if mode != c.ifOutdated || key != c.ifOutdatedKey {
			t.Errorf("--if-outdated, %s: got (%v, %q), want (%v, %q)", c.name, mode, key, c.ifOutdated, c.ifOutdatedKey)
		}
		mode, key = conditionalHelperInstallAction(c.res, c.err, true)
		if mode != c.ifNeeded || key != c.ifNeededKey {
			t.Errorf("--if-needed, %s: got (%v, %q), want (%v, %q)", c.name, mode, key, c.ifNeeded, c.ifNeededKey)
		}
	}
	for _, key := range []string{"service.install.nothing_to_update", "service.install.up_to_date"} {
		if i18n.T(i18n.ES, key) == key || i18n.T(i18n.EN, key) == key {
			t.Errorf("%s not translated", key)
		}
	}
	for _, name := range []string{"if-outdated", "if-needed"} {
		if f := serviceInstallCmd.Flags().Lookup(name); f == nil || !f.Hidden {
			t.Errorf("--%s must exist and stay hidden (installer plumbing)", name)
		}
	}
}

// A plain `service install` never asks for permission for a helper that is
// already current (ADR-0046); everything else still installs or updates,
// so every "what to do: kongtrol service install" hint keeps working.
func TestHelperInstallAction(t *testing.T) {
	cur := privops.PingResult{Protocol: privops.ProtocolVersion, Version: "1.2.0"}
	cases := []struct {
		name  string
		res   privops.PingResult
		err   error
		tools bool
		want  helperInstallMode
	}{
		{"missing", privops.PingResult{}, privops.ErrHelperNotInstalled, false, helperModeInstall},
		{"denied", privops.PingResult{}, privops.ErrDenied, false, helperModeInstall},
		{"client reports outdated", privops.PingResult{}, privops.ErrHelperOutdated, false, helperModeUpdate},
		{"older protocol", privops.PingResult{Protocol: privops.ProtocolVersion - 1, Version: "1.2.0"}, nil, false, helperModeUpdate},
		{"other version", privops.PingResult{Protocol: privops.ProtocolVersion, Version: "1.1.0"}, nil, false, helperModeUpdate},
		{"current, tools changed", cur, nil, true, helperModeInstall},
		{"current", cur, nil, false, helperModeCurrent},
	}
	for _, c := range cases {
		if got := helperInstallAction(c.res, c.err, "1.2.0", c.tools); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	for _, key := range []string{"service.install.updating", "service.install.updated", "service.install.current", "service.install.current_next"} {
		if i18n.T(i18n.ES, key) == key || i18n.T(i18n.EN, key) == key {
			t.Errorf("%s not translated", key)
		}
	}
}
