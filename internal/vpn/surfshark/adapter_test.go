package surfshark

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func TestRegistered(t *testing.T) {
	a, err := vpn.New("surfshark")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Capabilities().ManualReconnect {
		t.Error("surfshark must be ManualReconnect (no CLI)")
	}
	if _, ok := a.(vpn.AttachReporter); !ok {
		t.Error("surfshark must report pending user actions")
	}
}

func TestMatcher(t *testing.T) {
	cases := []struct {
		ifc  appattach.Iface
		want bool
	}{
		{appattach.Iface{Name: "Ethernet 4", Description: "Surfshark DCO adapter", Up: true}, true},
		{appattach.Iface{Name: "surfshark_wg", Up: true}, true},
		{appattach.Iface{Name: "surfshark_wg", Up: false}, false},
		{appattach.Iface{Name: "wlan0", Up: true}, false},
	}
	for _, c := range cases {
		if got := matcher.Match(c.ifc); got != c.want {
			t.Errorf("Match(%+v) = %v, want %v", c.ifc, got, c.want)
		}
	}
}
