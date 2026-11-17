package nordvpn

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func extra(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func TestWindowsConnectArgs(t *testing.T) {
	cases := []struct {
		extra map[string]string
		want  string
	}{
		// Legacy `server:` values keep their old meaning.
		{extra(), "-c"},
		{extra("server", "fastest"), "-c"},
		{extra("server", " Quick "), "-c"},
		{extra("server", "United States"), "-c -g United States"},
		{extra("server", "Dedicated IP"), "-c -g Dedicated IP"},
		{extra("server", "United States #3710"), "-c -n United States #3710"},
		{extra("server", "us3710"), "-c -n us3710"},
		{extra("server", "us3710.nordvpn.com"), "-c -n us3710.nordvpn.com"},
		// Standard fields.
		{extra("country", "Germany"), "-c -g Germany"},
		{extra("country", "Germany", "city", "Berlin"), "-c -g Berlin"},
		{extra("country", "Germany", "group", "P2P"), "-c -g P2P"},
	}
	for _, c := range cases {
		if got := strings.Join(windowsConnectArgs(targetFrom(c.extra)), " "); got != c.want {
			t.Errorf("windowsConnectArgs(%v) = %q, want %q", c.extra, got, c.want)
		}
	}
	if got := strings.Join(windowsDisconnectArgs(), " "); got != "-d" {
		t.Errorf("windowsDisconnectArgs = %q", got)
	}
}

func TestLinuxConnectArgs(t *testing.T) {
	cases := []struct {
		extra map[string]string
		want  string
	}{
		{extra(), "connect"},
		{extra("server", "fastest"), "connect"},
		{extra("server", "US"), "connect US"},
		{extra("server", "United States"), "connect United_States"},
		{extra("server", "  Dedicated   IP "), "connect Dedicated_IP"},
		{extra("server", "P2P"), "connect P2P"},
		{extra("server", "us3710"), "connect us3710"},
		{extra("server", "US3710.nordvpn.com"), "connect us3710"},
		{extra("country", "United States"), "connect United_States"},
		{extra("country", "Germany", "city", "Frankfurt am Main"), "connect Germany Frankfurt_am_Main"},
		{extra("city", "Berlin"), "connect Berlin"},
		{extra("group", "P2P", "country", "Netherlands"), "connect --group P2P Netherlands"},
	}
	for _, c := range cases {
		if got := strings.Join(linuxConnectArgs(targetFrom(c.extra)), " "); got != c.want {
			t.Errorf("linuxConnectArgs(%v) = %q, want %q", c.extra, got, c.want)
		}
	}
}

func TestParseLinuxStatus(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want linuxStatus
	}{
		{"current", "Status: Connected\nHostname: us3710.nordvpn.com\nIP: 1.2.3.4\nCountry: United States\nCurrent technology: NORDLYNX\n",
			linuxStatus{true, "us3710.nordvpn.com", "1.2.3.4"}},
		{"spinner+crlf", "\r-\r  \rStatus: Connected\r\nCurrent server: de100.nordvpn.com\r\nYour new IP: 5.6.7.8\r\n",
			linuxStatus{true, "de100.nordvpn.com", "5.6.7.8"}},
		{"disconnected", "Status: Disconnected\n", linuxStatus{}},
		{"lowercase keys", "status: connected\nserver ip: 9.9.9.9\n", linuxStatus{true, "", "9.9.9.9"}},
		{"garbage", "A new version of NordVPN is available!\n", linuxStatus{}},
	}
	for _, c := range cases {
		if got := parseLinuxStatus(c.raw); got != c.want {
			t.Errorf("%s: parseLinuxStatus = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestIsNotLoggedIn(t *testing.T) {
	for out, want := range map[string]bool{
		"You are not logged in.":            true,
		"Please log in to use the app.":     true,
		"Connecting to United States #3710": false,
		"":                                  false,
	} {
		if got := isNotLoggedIn(out); got != want {
			t.Errorf("isNotLoggedIn(%q) = %v, want %v", out, got, want)
		}
	}
}

func TestMatcher(t *testing.T) {
	cases := []struct {
		name, desc string
		want       bool
	}{
		{"NordLynx", "", true},
		{"nordlynx", "", true},
		{"NordVPN", "", true},
		{"Ethernet 3", "TAP-NordVPN Windows Adapter V9", true},
		{"nordtun", "", true},
		{"Ethernet", "Intel(R) Ethernet Connection", false},
		{"ProtonVPN", "ProtonVPN Tunnel", false},
		{"wg0", "", false},
	}
	for _, c := range cases {
		if got := matcher.Match(appattach.Iface{Name: c.name, Description: c.desc, Up: true}); got != c.want {
			t.Errorf("matcher.Match(%q,%q) = %v, want %v", c.name, c.desc, got, c.want)
		}
	}
}

func TestCapabilities(t *testing.T) {
	a := &Adapter{}
	a.Configure(vpn.AdapterConfig{Extra: extra("auto_connect", "false")})
	if !a.Capabilities().ManualReconnect {
		t.Error("auto_connect: false must make the watchdog notify instead of reconnecting")
	}
	a.Configure(vpn.AdapterConfig{})
	if got := a.Capabilities().ManualReconnect; got != (CheckPlatform() != nil) {
		t.Errorf("ManualReconnect = %v with CheckPlatform() = %v", got, CheckPlatform())
	}
}

func TestStatusZeroValueNormalized(t *testing.T) {
	if _, up := (&appattach.Session{}).Detect(matcher); up {
		t.Skip("a NordVPN tunnel is up on this machine")
	}
	if st := (&Adapter{}).Status(); st != vpn.StatusDisconnected {
		t.Fatalf("zero status = %q, want disconnected", st)
	}
}
