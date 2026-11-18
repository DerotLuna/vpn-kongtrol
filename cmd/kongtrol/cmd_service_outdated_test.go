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
