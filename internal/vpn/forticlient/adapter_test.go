package forticlient

import (
	"net"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func TestMatcher(t *testing.T) {
	v4 := []net.IP{net.ParseIP("10.212.134.5")}
	cases := []struct {
		ifc  appattach.Iface
		want bool
	}{
		{appattach.Iface{Name: "Ethernet 2", Description: "Fortinet SSL VPN Virtual Ethernet Adapter", Up: true, Addrs: v4}, true},
		{appattach.Iface{Name: "Fortinet SSL VPN", Up: true, Addrs: v4}, true},
		{appattach.Iface{Name: "fortissl0", Up: true, Addrs: v4}, true},
		{appattach.Iface{Name: "ppp0", Up: true, Addrs: v4}, true},
		// FortiClient keeps its adapter while disconnected: no IPv4 → not up.
		{appattach.Iface{Name: "Ethernet 2", Description: "Fortinet Virtual Ethernet Adapter (NDIS 6.30)", Up: true}, false},
		{appattach.Iface{Name: "eth0", Up: true, Addrs: v4}, false},
	}
	for _, c := range cases {
		if got := matcher.Match(c.ifc); got != c.want {
			t.Errorf("Match(%+v) = %v, want %v", c.ifc, got, c.want)
		}
	}
}

func TestCapabilities(t *testing.T) {
	a := &Adapter{}
	if a.Capabilities().ManualReconnect {
		t.Error("GUI automation / CLI is an automatic reconnect path")
	}
	a.Configure(vpn.AdapterConfig{Password: "x", Extra: map[string]string{"auto_connect": "false"}})
	if !a.Capabilities().ManualReconnect {
		t.Error("auto_connect: false must be ManualReconnect")
	}
	if a.lastCfg.Password != "" {
		t.Error("Configure must not keep a password")
	}
}
