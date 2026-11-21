package doctor

import (
	"net"
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func TestDoctorReportsActiveCiscoOverlap(t *testing.T) {
	stubOS(t, true)
	listNetworkInterfaces = func() []appattach.Iface {
		return []appattach.Iface{
			{Name: "VPN", Description: "Cisco AnyConnect Secure Mobility Client", Up: true, Addrs: []net.IP{net.ParseIP("10.0.0.2")}},
			{Name: "Cloudflare WARP", Up: true, Addrs: []net.IP{net.ParseIP("192.0.2.4")}},
			{Name: "TunnelBear", Description: "TunnelBear Adapter v9", Up: false},
		}
	}
	d := &run{opt: Options{Lang: i18n.EN}}
	d.checkVPNCoexistence(nil)
	if len(d.rep.Checks) != 1 || d.rep.Warnings != 1 || d.rep.Checks[0].Detail != i18n.T(i18n.EN, "doctor.coexistence.cisco") {
		t.Fatalf("diagnosis = %+v", d.rep)
	}
}

func TestCompetingDefaultRoutes(t *testing.T) {
	active := map[string]string{"warp": "cloudflarewarp", "proton": "protonvpn", "tail": "tailscale"}
	route := func(iface, cidr string) routing.Route {
		t.Helper()
		_, dst, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatal(err)
		}
		return routing.Route{Destination: *dst, Interface: iface}
	}
	routes := []routing.Route{
		route("ethernet", "0.0.0.0/0"),
		route("wi-fi", "0.0.0.0/0"),
		route("tail", "100.64.0.0/10"),
		route("warp", "0.0.0.0/0"),
		route("warp", "::/0"),
		route("proton", "0.0.0.0/1"),
		route("proton", "8000::/1"),
	}
	if got := competingDefaultRoutes(active, routes); len(got) != 0 {
		t.Fatalf("reported a conflict on a single full-route VPN: %+v", got)
	}
	routes = append(routes, route("proton", "128.0.0.0/1"), route("proton", "::/1"))
	want := []defaultConflict{
		{"IPv4", []string{"cloudflarewarp (warp)", "protonvpn (proton)"}},
		{"IPv6", []string{"cloudflarewarp (warp)", "protonvpn (proton)"}},
	}
	if got := competingDefaultRoutes(active, routes); !reflect.DeepEqual(got, want) {
		t.Fatalf("conflicts = %+v, want %+v", got, want)
	}
}
