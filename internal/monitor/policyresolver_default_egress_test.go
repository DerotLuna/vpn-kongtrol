package monitor

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
)

func TestDefaultEgressPreservesAllowedIPsAndExistingRoutes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wg0.conf")
	const conf = `[Interface]
Address = 10.7.0.2/32, fd43:1::2/128
DNS = 10.7.0.1
[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
Endpoint = 192.0.2.5:51820
AllowedIPs = 0.0.0.0/0, ::/0
`
	if err := os.WriteFile(path, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	lan := mustRoute("192.168.0.0/24", "lan0", "")
	rm := newFakeRoutes(lan)
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{
			"native": {Type: "wireguard", DefaultEgress: true},
			"office": {Type: "openvpn"},
		},
		Policies: []config.PolicyRule{
			{Name: "native", Via: "native", Match: config.MatchSpec{IPRanges: []string{"198.51.100.0/24", "2001:db8:1::/48"}}},
			{Name: "office", Via: "office", Match: config.MatchSpec{IPRanges: []string{"198.51.100.42/32", "2001:db8:1:2::/64"}}},
		},
	}
	r := testResolver(t, cfg, rm, nil)
	r.wgSet = func(string, string, []string, string) error {
		t.Fatal("rewrote AllowedIPs for native WireGuard")
		return nil
	}
	if err := r.RegisterProfile("native", "wg0", path); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterTunnel("office", "tun0", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"198.51.100.0/24|wg0", "198.51.100.42/32|tun0", "2001:db8:1:2::/64|tun0", "2001:db8:1::/48|wg0"}
	if got := rm.keys(); !slices.Equal(got, want) {
		t.Fatalf("policy routes = %v, want %v", got, want)
	}
	r.UnregisterProfile("native")
	all, err := rm.List()
	if err != nil || !slices.ContainsFunc(all, func(route routing.Route) bool { return key(route) == key(lan) }) {
		t.Fatalf("existing LAN route missing: %v, %v", all, err)
	}
}
