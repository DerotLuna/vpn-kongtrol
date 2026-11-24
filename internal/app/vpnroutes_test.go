package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type routeReportingAdapter struct {
	fakeAdapter
	routes *vpn.TunnelRoutes
}

func (r *routeReportingAdapter) TunnelRoutes() *vpn.TunnelRoutes { return r.routes }

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func byName(list []ProfileOwnRoutes) map[string]ProfileOwnRoutes {
	out := map[string]ProfileOwnRoutes{}
	for _, p := range list {
		out[p.Name] = p
	}
	return out
}

func dests(p ProfileOwnRoutes) map[string]string {
	out := map[string]string{}
	for _, r := range p.Routes {
		out[r.Destination] = r.Origin
	}
	return out
}

func TestOwnRoutes_FileServerAndApps(t *testing.T) {
	ovpn := writeTemp(t, "work.ovpn", "client\nremote vpn.example 1194\nroute 10.20.0.0 255.255.0.0\n"+
		"route 192.168.50.0 255.255.255.0 net_gateway\nroute intranet.example\n# route 1.1.1.1\n")
	wgSplit := writeTemp(t, "home.conf", "[Interface]\nAddress = 10.7.0.2/32\n[Peer]\nAllowedIPs = 0.0.0.0/0, ::/0\n")
	wgFull := writeTemp(t, "full.conf", "[Interface]\nAddress = 10.8.0.2/32\n[Peer]\nAllowedIPs = 0.0.0.0/1, 128.0.0.0/1\n")
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{
		"work":    {Type: "openvpn", ConfigFile: ovpn},
		"idle":    {Type: "openvpn", ConfigFile: ovpn},
		"home":    {Type: "wireguard", ConfigFile: wgSplit},
		"full":    {Type: "wireguard", ConfigFile: wgFull, DefaultEgress: true},
		"proton":  {Type: "protonvpn"},
		"forti":   {Type: "forticlient"},
		"missing": {Type: "openvpn", ConfigFile: filepath.Join(t.TempDir(), "gone.ovpn")},
	}}
	work := &routeReportingAdapter{
		fakeAdapter: fakeAdapter{status: vpn.StatusConnected},
		routes:      &vpn.TunnelRoutes{Origin: vpn.OriginServer, Routes: []string{"10.30.0.0/16", "10.20.0.0/16"}, AllTraffic: true},
	}
	set := vpn.NewSet(map[string]vpn.VPNAdapter{"work": work, "idle": &fakeAdapter{status: vpn.StatusDisconnected}})

	got := byName(OwnRoutes(cfg, set, ""))

	w := got["work"]
	if !w.Connected || w.ServerPending || w.Source != OwnSourceConfig || !w.AllTraffic {
		t.Fatalf("work = %+v", w)
	}
	want := map[string]string{"10.20.0.0/16": OwnRouteFile, "10.30.0.0/16": OwnRouteServer, "0.0.0.0/0": OwnRouteServer}
	if !reflect.DeepEqual(dests(w), want) {
		t.Errorf("work routes = %v, want %v (net_gateway and hostnames skipped, file wins a duplicate)", dests(w), want)
	}
	if i := got["idle"]; i.Connected || !i.ServerPending || len(i.Routes) != 1 {
		t.Errorf("idle = %+v, want file route only and server_pending", i)
	}
	if h := got["home"]; !h.RulesOnly || h.AllTraffic || len(h.Routes) != 2 {
		t.Errorf("home = %+v, want rules_only (AllowedIPs replaced) and not all traffic", h)
	}
	if f := got["full"]; f.RulesOnly || !f.AllTraffic {
		t.Errorf("full = %+v, want all traffic with default_egress", f)
	}
	if p := got["proton"]; p.Source != OwnSourceApp || !p.AllTraffic {
		t.Errorf("proton = %+v", p)
	}
	if f := got["forti"]; f.Source != OwnSourceApp || f.AllTraffic {
		t.Errorf("forti = %+v, want app, all traffic unknown", f)
	}
	if m := got["missing"]; m.Source != OwnSourceUnreadable {
		t.Errorf("missing = %+v", m)
	}
	if only := OwnRoutes(cfg, set, "home"); len(only) != 1 || only[0].Name != "home" {
		t.Errorf("only home = %+v", only)
	}
	if none := OwnRoutes(cfg, nil, "work"); none[0].Connected || !none[0].ServerPending {
		t.Errorf("without daemon = %+v", none[0])
	}
}

func TestOwnRoutes_Conflicts(t *testing.T) {
	ovpn := writeTemp(t, "work.ovpn", "client\nroute 10.0.0.0 255.0.0.0\nroute 172.16.5.0 255.255.255.0\nredirect-gateway def1\n")
	wg := writeTemp(t, "home.conf", "[Interface]\nAddress = 10.7.0.2/32\n[Peer]\nAllowedIPs = 10.0.0.0/8\n")
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{
			"work":  {Type: "openvpn", ConfigFile: ovpn},
			"home":  {Type: "wireguard", ConfigFile: wg},
			"other": {Type: "wireguard", ConfigFile: wg},
		},
		Policies: []config.PolicyRule{
			{Name: "db", Via: "other", Match: config.MatchSpec{IPRanges: []string{"10.1.2.3"}}},
			{Name: "lan", Via: "other", Match: config.MatchSpec{IPRanges: []string{"172.16.0.0/16"}}},
			{Name: "same", Via: "other", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}},
			{Name: "mine", Via: "work", Match: config.MatchSpec{IPRanges: []string{"10.9.0.0/16"}}},
			{Name: "paused", Via: "other", Disabled: true, Match: config.MatchSpec{IPRanges: []string{"10.5.0.0/16"}}},
			{Name: "web", Via: "other", Match: config.MatchSpec{Domains: []string{"example.com"}}},
		},
	}
	got := byName(OwnRoutes(cfg, nil, ""))
	conflicts := map[string][]RouteConflict{}
	for _, r := range got["work"].Routes {
		conflicts[r.Destination] = r.Conflicts
	}
	want10 := []RouteConflict{
		{Rule: "db", Via: "other", Destination: "10.1.2.3/32", Winner: WinnerRule},
		{Rule: "same", Via: "other", Destination: "10.0.0.0/8", Winner: WinnerSystem},
	}
	if !reflect.DeepEqual(conflicts["10.0.0.0/8"], want10) {
		t.Errorf("10.0.0.0/8 conflicts = %+v, want %+v (own-profile and paused rules ignored)", conflicts["10.0.0.0/8"], want10)
	}
	want172 := []RouteConflict{{Rule: "lan", Via: "other", Destination: "172.16.0.0/16", Winner: WinnerVPN}}
	if !reflect.DeepEqual(conflicts["172.16.5.0/24"], want172) {
		t.Errorf("172.16.5.0/24 conflicts = %+v, want %+v", conflicts["172.16.5.0/24"], want172)
	}
	if c := conflicts["0.0.0.0/0"]; len(c) != 0 {
		t.Errorf("a full tunnel is never in conflict (narrower rules always win), got %+v", c)
	}
	for _, r := range got["home"].Routes {
		if len(r.Conflicts) != 0 {
			t.Errorf("rules_only WireGuard routes are not used, so no conflicts: %+v", r)
		}
	}
}

func TestCarriesAllTraffic(t *testing.T) {
	r := func(ss ...string) []OwnRoute {
		var out []OwnRoute
		for _, s := range ss {
			out = append(out, OwnRoute{Destination: s})
		}
		return out
	}
	for _, c := range []struct {
		routes []OwnRoute
		want   bool
	}{
		{r("0.0.0.0/0"), true},
		{r("0.0.0.0/1", "128.0.0.0/1"), true},
		{r("0.0.0.0/1"), false},
		{r("::/0"), false},
		{r("10.0.0.0/8"), false},
	} {
		if got := CarriesAllTraffic(c.routes); got != c.want {
			t.Errorf("%v: got %v", c.routes, got)
		}
	}
}
