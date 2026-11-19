package main

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// The Windows installer runs `service install --if-outdated` on every
// upgrade: only a helper speaking an older protocol may trigger UAC.
func TestOutdatedHelperAction(t *testing.T) {
	cases := []struct {
		name    string
		res     privops.PingResult
		err     error
		install bool
		key     string
	}{
		{"not installed", privops.PingResult{}, privops.ErrHelperNotInstalled, false, "service.install.nothing_to_update"},
		{"current protocol", privops.PingResult{Protocol: privops.ProtocolVersion}, nil, false, "service.install.up_to_date"},
		{"older protocol", privops.PingResult{Protocol: privops.ProtocolVersion - 1}, nil, true, ""},
		// The client turns a reply with another protocol into
		// ErrHelperOutdated: that helper must be updated, not skipped.
		{"client reports outdated", privops.PingResult{}, privops.ErrHelperOutdated, true, ""},
	}
	for _, c := range cases {
		install, key := outdatedHelperAction(c.res, c.err)
		if install != c.install || key != c.key {
			t.Errorf("%s: got (%v, %q), want (%v, %q)", c.name, install, key, c.install, c.key)
		}
		if key != "" && (i18n.T(i18n.ES, key) == key || i18n.T(i18n.EN, key) == key) {
			t.Errorf("%s: %s not translated", c.name, key)
		}
	}
	f := serviceInstallCmd.Flags().Lookup("if-outdated")
	if f == nil || !f.Hidden {
		t.Fatal("--if-outdated must exist and stay hidden (installer plumbing)")
	}
}

// A plain `service install` never asks for permission for a helper that is
// already current (ADR-0045); everything else still installs or updates,
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
