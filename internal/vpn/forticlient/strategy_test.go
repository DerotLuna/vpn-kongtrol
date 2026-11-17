package forticlient

import (
	"slices"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want Version
		ok   bool
	}{
		{"6.4.10.1821", Version{6, 4, 10, 1821}, true},
		{"7.2", Version{7, 2, 0, 0}, true},
		{" v7.0.3 ", Version{7, 0, 3, 0}, true},
		{"", Version{}, false},
		{"6.x", Version{}, false},
	}
	for _, c := range cases {
		got, ok := ParseVersion(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("ParseVersion(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

// One case per row of docs/forticlient-support.md.
func TestSelectPlan(t *testing.T) {
	fv := `C:\Program Files\Fortinet\FortiClient\FortiVPN.exe`
	cases := []struct {
		name string
		goos string
		in   Install
		want Plan
	}{
		{"6.4.10 verified", "windows", Install{Version: "6.4.10.1821"}, Plan{StrategyGUI64, SupportVerified}},
		{"other 6.4 build", "windows", Install{Version: "6.4.10.9999"}, Plan{StrategyGUI64, SupportVerified}},
		{"6.4.9", "windows", Install{Version: "6.4.9.1557"}, Plan{StrategyGUI64, SupportUntested}},
		{"6.2", "windows", Install{Version: "6.2.9.0"}, Plan{StrategyGUI64, SupportUntested}},
		{"6.0", "windows", Install{Version: "6.0.10.0"}, Plan{StrategyGUI64, SupportUntested}},
		{"unknown version", "windows", Install{}, Plan{StrategyGUI64, SupportUntested}},
		{"7.x with FortiVPN.exe", "windows", Install{Version: "7.2.4.0972", FortiVPN: fv}, Plan{StrategyFortiVPN, SupportReported}},
		{"FortiVPN.exe wins over version", "windows", Install{Version: "6.4.10.1821", FortiVPN: fv}, Plan{StrategyFortiVPN, SupportReported}},
		{"7.x without FortiVPN.exe", "windows", Install{Version: "7.0.8.0427"}, Plan{StrategyAttach, SupportUntested}},
		{"7.4", "windows", Install{Version: "7.4.7.2003"}, Plan{StrategyAttach, SupportUntested}},
		{"5.x", "windows", Install{Version: "5.6.6.1167"}, Plan{StrategyAttach, SupportUntested}},
		{"linux", "linux", Install{Version: "7.4.4"}, Plan{StrategyCLI, SupportUntested}},
		{"macos", "darwin", Install{}, Plan{StrategyCLI, SupportUntested}},
	}
	for _, c := range cases {
		if got := SelectPlan(c.goos, c.in); got != c.want {
			t.Errorf("%s: SelectPlan = %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Credentials must never be FortiVPN.exe arguments: any local process can
// read another process's command line.
func TestFortiVPNArgs_NoCredentials(t *testing.T) {
	if got, want := fortiVPNConnectArgs("Office VPN"), []string{"--cli", "--connect", "--tunnel", "Office VPN"}; !slices.Equal(got, want) {
		t.Errorf("connect args = %q, want %q", got, want)
	}
	if got, want := fortiVPNDisconnectArgs(""), []string{"--cli", "--disconnect"}; !slices.Equal(got, want) {
		t.Errorf("disconnect args = %q, want %q", got, want)
	}
	for _, a := range append(fortiVPNConnectArgs("x"), fortiVPNDisconnectArgs("x")...) {
		l := strings.ToLower(a)
		if strings.Contains(l, "pass") || strings.Contains(l, "user") || a == "-u" || a == "-p" {
			t.Errorf("credential-like argument %q", a)
		}
	}
}
