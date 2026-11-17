package api

import (
	"net"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func mustRoute(t *testing.T, cidr, gw, iface string) routing.Route {
	t.Helper()
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatal(err)
	}
	return routing.Route{Destination: *n, Gateway: net.ParseIP(gw), Interface: iface}
}

func TestBuildTunnelRoutes_GroupsByOrigin(t *testing.T) {
	const ifc = "Local Area Connection"
	os := []routing.Route{
		mustRoute(t, "10.200.0.0/24", "192.168.220.249", ifc),
		mustRoute(t, "192.168.220.0/24", "192.168.220.249", ifc),
		mustRoute(t, "192.168.220.248/30", "", ifc), // p2p subnet: noise
		mustRoute(t, "192.168.220.250/32", "", ifc), // own address: noise
		mustRoute(t, "192.168.220.251/32", "", ifc), // broadcast: noise
		mustRoute(t, "224.0.0.0/4", "", ifc),
		mustRoute(t, "255.255.255.255/32", "", ifc),
		mustRoute(t, "172.16.5.0/24", "192.168.220.249", ifc), // client config route
		mustRoute(t, "10.50.0.0/16", "192.168.220.249", ifc),  // policy
		mustRoute(t, "0.0.0.0/0", "192.168.2.1", "Wi-Fi"),     // other interface
	}
	reported := &vpn.TunnelRoutes{Origin: vpn.OriginServer, Routes: []string{"192.168.220.0/24", "10.200.0.0/24", "10.200.1.0/24"}}
	dto := buildTunnelRoutes("ekiipago-ovpn", []string{"OpenVPN Data Channel Offload", ifc},
		net.ParseIP("192.168.220.250"), os, reported, map[string]string{"10.50.0.0/16": "oficina"})

	want := []tunnelRouteDTO{
		{Destination: "10.200.0.0/24", Origin: "server", Gateway: "192.168.220.249"},
		{Destination: "10.200.1.0/24", Origin: "server"},
		{Destination: "192.168.220.0/24", Origin: "server", Gateway: "192.168.220.249"},
		{Destination: "10.50.0.0/16", Origin: "policy", Policy: "oficina", Gateway: "192.168.220.249"},
		{Destination: "172.16.5.0/24", Origin: "vpn", Gateway: "192.168.220.249"},
	}
	if len(dto.Routes) != len(want) {
		t.Fatalf("routes = %+v", dto.Routes)
	}
	for i := range want {
		if dto.Routes[i] != want[i] {
			t.Fatalf("route %d = %+v, want %+v", i, dto.Routes[i], want[i])
		}
	}
	if dto.Counts["server"] != 3 || dto.Counts["policy"] != 1 || dto.Counts["vpn"] != 1 || dto.Total != 5 {
		t.Fatalf("counts = %v total %d", dto.Counts, dto.Total)
	}
	if dto.AllTraffic || dto.Interface != ifc {
		t.Fatalf("all=%v iface=%q", dto.AllTraffic, dto.Interface)
	}
}

func TestBuildTunnelRoutes_AllTraffic(t *testing.T) {
	os := []routing.Route{
		mustRoute(t, "0.0.0.0/1", "10.8.0.1", "tun0"),
		mustRoute(t, "128.0.0.0/1", "10.8.0.1", "tun0"),
	}
	dto := buildTunnelRoutes("p", []string{"tun0"}, net.ParseIP("10.8.0.2"), os, nil, nil)
	if !dto.AllTraffic || dto.Total != 0 {
		t.Fatalf("def1 halves: %+v", dto)
	}
	dto = buildTunnelRoutes("p", nil, nil, nil, &vpn.TunnelRoutes{Origin: vpn.OriginServer, AllTraffic: true}, nil)
	if !dto.AllTraffic {
		t.Fatal("reported redirect-gateway lost")
	}
}

func TestRestViaAndPolicyNames(t *testing.T) {
	routes := []routing.Route{
		mustRoute(t, "0.0.0.0/0", "10.8.0.1", "tun0"),
		mustRoute(t, "0.0.0.0/0", "192.168.2.1", "Wi-Fi"),
	}
	if got := restVia(routes, map[string]bool{"tun0": true}); got != "Wi-Fi" {
		t.Fatalf("restVia = %q", got)
	}
	_, n, _ := net.ParseCIDR("10.50.0.0/16")
	rules := []policy.Rule{
		{Name: "otra", Via: "x", Match: policy.MatchSpec{IPRanges: []*net.IPNet{n}}},
		{Name: "dominios", Via: "p", Match: policy.MatchSpec{Domains: []string{"*.corp"}}},
		{Name: "oficina", Via: "p", Match: policy.MatchSpec{IPRanges: []*net.IPNet{n}}},
	}
	got := policyNames("p", []string{"10.50.0.0/16", "10.9.9.9/32"}, rules)
	if got["10.50.0.0/16"] != "oficina" || got["10.9.9.9/32"] != "dominios" {
		t.Fatalf("policyNames = %v", got)
	}
}
