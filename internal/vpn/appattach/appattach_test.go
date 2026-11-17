package appattach

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestGlob(t *testing.T) {
	cases := []struct {
		pat, s string
		want   bool
	}{
		{"*nordlynx*", "NordLynx", true},
		{"nordtun*", "nordtun0", true},
		{"nordtun*", "xnordtun0", false},
		{"utun?", "utun4", true},
		{"utun?", "utun42", false},
		{"*", "", true},
		{"*fortinet*", "Fortinet SSL VPN Virtual Ethernet Adapter", true},
		{"proton*", "pvpn0", false},
		{"a*b*c", "aXXbYYc", true},
		{"a*b*c", "aXXbYY", false},
	}
	for _, c := range cases {
		if got := Glob(c.pat, c.s); got != c.want {
			t.Errorf("Glob(%q,%q) = %v, want %v", c.pat, c.s, got, c.want)
		}
	}
}

func ip(s string) net.IP { return net.ParseIP(s) }

func TestFind(t *testing.T) {
	ifaces := []Iface{
		{Name: "Ethernet", Description: "Intel(R) Ethernet", Up: true, Addrs: []net.IP{ip("192.168.1.5")}},
		{Name: "pvpnksintrf0", Up: true, Addrs: []net.IP{ip("100.85.0.1")}},
		{Name: "Ethernet 3", Description: "TAP-NordVPN Windows Adapter V9", Up: true, Addrs: []net.IP{ip("fe80::1"), ip("10.8.3.4")}},
		{Name: "NordLynx", Description: "NordLynx Tunnel", Up: false, Addrs: []net.IP{ip("10.5.0.2")}},
		{Name: "proton0", Up: true, Addrs: []net.IP{ip("fe80::2")}},
		{Name: "fortissl0", Up: true},
	}
	cases := []struct {
		name     string
		m        Matcher
		wantOK   bool
		wantName string
		wantIP   string
	}{
		{"by description", Matcher{Patterns: []string{"*nordvpn*", "*nordlynx*"}}, true, "Ethernet 3", "10.8.3.4"},
		{"down iface skipped", Matcher{Patterns: []string{"nordlynx"}}, false, "", ""},
		{"exclude kill-switch dummy", Matcher{Patterns: []string{"pvpn*", "proton*"}, Exclude: []string{"*intrf*"}}, true, "proton0", "fe80::2"},
		{"require ipv4", Matcher{Patterns: []string{"fortissl*"}, RequireIPv4: true}, false, "", ""},
		{"no ipv4 required", Matcher{Patterns: []string{"fortissl*"}}, true, "fortissl0", ""},
		{"override", Matcher{Patterns: []string{"*nordvpn*"}}.WithOverride(" Ethernet ,x"), true, "Ethernet", "192.168.1.5"},
		{"empty override keeps patterns", Matcher{Patterns: []string{"proton0"}}.WithOverride(" "), true, "proton0", "fe80::2"},
	}
	for _, c := range cases {
		got, ok := Find(ifaces, c.m)
		if ok != c.wantOK || got.Name != c.wantName {
			t.Errorf("%s: Find = %+v,%v; want %q,%v", c.name, got, ok, c.wantName, c.wantOK)
			continue
		}
		if c.wantIP == "" && got.IP != nil {
			t.Errorf("%s: IP = %v, want none", c.name, got.IP)
		}
		if c.wantIP != "" && !got.IP.Equal(ip(c.wantIP)) {
			t.Errorf("%s: IP = %v, want %s", c.name, got.IP, c.wantIP)
		}
	}
}

func TestParseOptions(t *testing.T) {
	cases := []struct {
		extra map[string]string
		want  Options
	}{
		{nil, Options{AutoConnect: true, WaitTimeout: DefaultWaitTimeout}},
		{map[string]string{"auto_connect": "false", "wait_timeout": "2m", "interface": " utun9 ", "app_path": "/x"},
			Options{AutoConnect: false, WaitTimeout: 2 * time.Minute, Interface: "utun9", AppPath: "/x"}},
		{map[string]string{"auto_connect": "true", "wait_timeout": "1s"}, Options{AutoConnect: true, WaitTimeout: DefaultWaitTimeout}},
		{map[string]string{"auto_connect": "garbage", "wait_timeout": "nope"}, Options{AutoConnect: true, WaitTimeout: DefaultWaitTimeout}},
	}
	for i, c := range cases {
		if got := ParseOptions(c.extra); got != c.want {
			t.Errorf("case %d: %+v, want %+v", i, got, c.want)
		}
	}
}

func TestParseNC(t *testing.T) {
	list := `Available network connection services in the current set (*=enabled):
* (Disconnected)   1111 VPN (com.example.a) "Other VPN"          [VPN:com.example]
* (Connected)      2222 VPN (com.nordvpn.macos.NEPacketTunnel) "NordVPN NordLynx" [VPN:com.nordvpn.macos]
`
	got := parseNCList(list)
	if len(got) != 1 || got[0] != "NordVPN NordLynx" {
		t.Fatalf("parseNCList = %v", got)
	}
	if name, ok := pickService(got, []string{"nordvpn*"}); !ok || name != "NordVPN NordLynx" {
		t.Errorf("pickService = %q,%v", name, ok)
	}
	if _, ok := pickService(got, []string{"surfshark*"}); ok {
		t.Error("pickService matched the wrong provider")
	}
	status := `Connected
Extended Status <dictionary> {
  IPv4 : <dictionary> {
    Addresses : <array> {
      0 : 10.5.0.2
    }
    InterfaceName : utun4
  }
  Status : 2
}`
	tun := parseNCStatus(status)
	if tun.Name != "utun4" || !tun.IP.Equal(ip("10.5.0.2")) {
		t.Errorf("parseNCStatus = %+v", tun)
	}
}

// fakeNet is a mutable interface list for session tests.
type fakeNet struct {
	mu sync.Mutex
	up bool
}

func (f *fakeNet) set(up bool) { f.mu.Lock(); f.up = up; f.mu.Unlock() }
func (f *fakeNet) list() []Iface {
	f.mu.Lock()
	defer f.mu.Unlock()
	return []Iface{{Name: "surfshark_wg", Up: f.up, Addrs: []net.IP{ip("10.14.0.2")}}}
}

func newTestSession(f *fakeNet, launched *int) *Session {
	return &Session{
		ifaces:  f.list,
		service: func([]string) (Tunnel, bool) { return Tunnel{}, false },
		launch:  func(App, string) error { *launched++; return nil },
		poll:    5 * time.Millisecond,
	}
}
