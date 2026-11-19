package main

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
)

// TestFortiDetectLine: the wizard shows the detected FortiClient and the
// strategy SelectPlan picks for it instead of asking the version (#16).
func TestFortiDetectLine(t *testing.T) {
	strategy := func(lang i18n.Lang, s forticlient.Strategy) string {
		return i18n.T(lang, "cli.doctor.forti.strategy."+string(s))
	}
	cases := []struct {
		name string
		goos string
		in   forticlient.Install
		want func(lang i18n.Lang) string
	}{
		{"7.4 with FortiVPN.exe", "windows",
			forticlient.Install{Path: `C:\FC\FortiClient.exe`, Version: "7.4.4.1234", FortiVPN: `C:\FC\FortiVPN.exe`},
			func(l i18n.Lang) string {
				return i18n.F(l, "wizard.forti.detected", "7.4.4.1234", strategy(l, forticlient.StrategyFortiVPN))
			}},
		{"6.4.10 GUI", "windows",
			forticlient.Install{Path: `C:\FC\FortiClient.exe`, Version: "6.4.10.1821"},
			func(l i18n.Lang) string {
				return i18n.F(l, "wizard.forti.detected", "6.4.10.1821", strategy(l, forticlient.StrategyGUI64))
			}},
		{"7.2 without FortiVPN.exe", "windows",
			forticlient.Install{Path: `C:\FC\FortiClient.exe`, Version: "7.2.0"},
			func(l i18n.Lang) string {
				return i18n.F(l, "wizard.forti.detected", "7.2.0", strategy(l, forticlient.StrategyAttach))
			}},
		{"version unreadable", "windows",
			forticlient.Install{Path: `C:\FC\FortiClient.exe`},
			func(l i18n.Lang) string {
				return i18n.F(l, "wizard.forti.version_unknown", strategy(l, forticlient.StrategyGUI64))
			}},
		{"not installed", "windows", forticlient.Install{},
			func(l i18n.Lang) string { return i18n.T(l, "wizard.forti.not_found") }},
		{"linux", "linux", forticlient.Install{Path: "/opt/forticlient/forticlient"},
			func(l i18n.Lang) string {
				return i18n.F(l, "wizard.forti.plan", strategy(l, forticlient.StrategyCLI))
			}},
	}
	for _, c := range cases {
		for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
			got := fortiDetectLine(lang, c.goos, c.in)
			if want := c.want(lang); got != want {
				t.Errorf("%s/%s: got %q, want %q", c.name, lang, got, want)
			}
			if got == "" || strings.Contains(got, "forti.") || strings.Contains(got, "%!") {
				t.Errorf("%s/%s: untranslated line %q", c.name, lang, got)
			}
		}
	}
}
