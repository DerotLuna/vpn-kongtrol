package privops

import (
	"errors"
	"net/netip"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateInterfaceName(t *testing.T) {
	ok := []string{"eth0", "wg0", "utun3", "tun_1", "en0.5"}
	if runtime.GOOS == "windows" {
		ok = append(ok, "Ethernet 2", "Wi-Fi", "Conexión de área local* 1", "OpenVPN TAP-Windows6")
	}
	for _, n := range ok {
		if err := ValidateInterfaceName(n); err != nil {
			t.Errorf("%q: unexpected error %v", n, err)
		}
	}
	bad := []string{"", " eth0", "eth0 ", "-eth0", "a=b", `a"b`, "a'b", "a;b", "a/b", `a\b`, "a\nb", "a\x00b",
		"a|b", "a&b", "a$b", "a`b", strings.Repeat("x", 300)}
	for _, n := range bad {
		if err := ValidateInterfaceName(n); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q: want ErrInvalidRequest, got %v", n, err)
		}
	}
}

func TestNormalizeRoute(t *testing.T) {
	p := netip.MustParsePrefix
	a := netip.MustParseAddr
	tests := []struct {
		name    string
		in      Route
		want    netip.Prefix
		wantErr bool
	}{
		{"ok", Route{Destination: p("10.0.0.0/8"), Interface: "wg0", Metric: 5}, p("10.0.0.0/8"), false},
		{"masked", Route{Destination: p("10.1.2.3/16"), Interface: "wg0"}, p("10.1.0.0/16"), false},
		{"gateway", Route{Destination: p("10.0.0.0/8"), Gateway: a("10.8.0.1"), Interface: "wg0"}, p("10.0.0.0/8"), false},
		{"v6", Route{Destination: p("fd00::/8"), Interface: "wg0"}, p("fd00::/8"), false},
		{"invalid prefix", Route{Interface: "wg0"}, netip.Prefix{}, true},
		{"loopback", Route{Destination: p("127.0.0.0/8"), Interface: "wg0"}, netip.Prefix{}, true},
		{"multicast", Route{Destination: p("224.0.0.0/4"), Interface: "wg0"}, netip.Prefix{}, true},
		{"family mismatch", Route{Destination: p("10.0.0.0/8"), Gateway: a("fe80::1"), Interface: "wg0"}, netip.Prefix{}, true},
		{"gateway unspecified", Route{Destination: p("10.0.0.0/8"), Gateway: a("0.0.0.0"), Interface: "wg0"}, netip.Prefix{}, true},
		{"metric negative", Route{Destination: p("10.0.0.0/8"), Interface: "wg0", Metric: -1}, netip.Prefix{}, true},
		{"metric high", Route{Destination: p("10.0.0.0/8"), Interface: "wg0", Metric: MaxMetric + 1}, netip.Prefix{}, true},
		{"bad iface", Route{Destination: p("10.0.0.0/8"), Interface: "wg0 metric=1"}, netip.Prefix{}, true},
	}
	for _, tc := range tests {
		got, err := NormalizeRoute(tc.in)
		if tc.wantErr {
			if !errors.Is(err, ErrInvalidRequest) {
				t.Errorf("%s: want ErrInvalidRequest, got %v", tc.name, err)
			}
			continue
		}
		if err != nil || got.Destination != tc.want {
			t.Errorf("%s: got %v %v, want %v", tc.name, got.Destination, err, tc.want)
		}
	}
}

func TestNormalizeDNS(t *testing.T) {
	a := netip.MustParseAddr
	got, err := NormalizeDNS(DNSRequest{Interface: "wg0", Servers: []netip.Addr{a("1.1.1.1"), a("1.1.1.1"), a("::ffff:9.9.9.9")}})
	if err != nil || len(got.Servers) != 2 || got.Servers[1] != a("9.9.9.9") {
		t.Fatalf("dedupe/unmap: %v %v", got.Servers, err)
	}
	for name, req := range map[string]DNSRequest{
		"empty":       {Interface: "wg0"},
		"unspecified": {Interface: "wg0", Servers: []netip.Addr{a("0.0.0.0")}},
		"multicast":   {Interface: "wg0", Servers: []netip.Addr{a("224.0.0.1")}},
		"zero addr":   {Interface: "wg0", Servers: []netip.Addr{{}}},
		"too many":    {Interface: "wg0", Servers: make([]netip.Addr, MaxDNSServers+1)},
		"bad iface":   {Interface: "a=b", Servers: []netip.Addr{a("1.1.1.1")}},
	} {
		if _, err := NormalizeDNS(req); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: want ErrInvalidRequest, got %v", name, err)
		}
	}
}

func TestNormalizeKillSwitchAndAllowedIPs(t *testing.T) {
	a := netip.MustParseAddr
	if _, err := NormalizeKillSwitch(KillSwitchRequest{Interfaces: []string{"wg0"}, Endpoints: []netip.Addr{a("203.0.113.5")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeKillSwitch(KillSwitchRequest{Endpoints: []netip.Addr{a("127.0.0.1")}}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("loopback endpoint accepted: %v", err)
	}
	if _, err := NormalizeKillSwitch(KillSwitchRequest{Interfaces: []string{"x;rm"}}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("bad interface accepted: %v", err)
	}
	key := "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
	good := WGAllowedIPsRequest{Interface: "wg0", PeerPublicKey: key, AllowedIPs: []netip.Prefix{netip.MustParsePrefix("10.1.2.3/24")}}
	got, err := NormalizeAllowedIPs(good)
	if err != nil || got.AllowedIPs[0].String() != "10.1.2.0/24" {
		t.Fatalf("allowed ips: %v %v", got.AllowedIPs, err)
	}
	for name, req := range map[string]WGAllowedIPsRequest{
		"bad key":  {Interface: "wg0", PeerPublicKey: "abc", AllowedIPs: good.AllowedIPs},
		"bad name": {Interface: "wg 0", PeerPublicKey: key, AllowedIPs: good.AllowedIPs},
		"empty":    {Interface: "wg0", PeerPublicKey: key},
		"invalid":  {Interface: "wg0", PeerPublicKey: key, AllowedIPs: []netip.Prefix{{}}},
	} {
		if _, err := NormalizeAllowedIPs(req); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: want ErrInvalidRequest, got %v", name, err)
		}
	}
}

func TestValidateConfigPathAndNames(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "work.ovpn")
	if err := ValidateConfigPath(abs, ".ovpn"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"", "work.ovpn", abs + ".exe", t.TempDir() + string(filepath.Separator) + ".." + string(filepath.Separator) + "x.ovpn", `\\server\share\x.ovpn`, "a\x00.ovpn"} {
		if err := ValidateConfigPath(p, ".ovpn"); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%q accepted: %v", p, err)
		}
	}
	for _, n := range []string{"wg0", "kongtrol-wg-1", "a.b_c=d+e"} {
		if err := ValidateWGName(n); err != nil {
			t.Errorf("wg name %q: %v", n, err)
		}
	}
	for _, n := range []string{"", "-x", "a b", "a/b", strings.Repeat("a", 33), "..x"} {
		if err := ValidateWGName(n); err == nil {
			t.Errorf("wg name %q accepted", n)
		}
	}
	for _, h := range []string{"intranet.corp", "a-b.example.com", "x"} {
		if err := ValidateHostname(h); err != nil {
			t.Errorf("host %q: %v", h, err)
		}
	}
	for _, h := range []string{"", "UPPER.com", "a..b", "-a.com", "a b", "a\nb", "a.com#x", strings.Repeat("a", 64) + ".com"} {
		if err := ValidateHostname(h); err == nil {
			t.Errorf("host %q accepted", h)
		}
	}
	if ValidateSessionID(randomHex(16)) != nil || ValidateSessionID("../x") == nil {
		t.Error("session id validation")
	}
}
