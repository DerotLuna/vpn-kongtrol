package security

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestParseKillSwitchSpec(t *testing.T) {
	exists := func(n string) bool { return n != "gone0" }
	req := parseKillSwitchSpec("wg0,gone0, tun1|1.2.3.4,bad,::ffff:5.6.7.8", true, exists)
	if len(req.Interfaces) != 2 || req.Interfaces[0] != "wg0" || req.Interfaces[1] != "tun1" {
		t.Fatalf("interfaces %v", req.Interfaces)
	}
	want := []netip.Addr{netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("5.6.7.8")}
	if len(req.Endpoints) != 2 || req.Endpoints[0] != want[0] || req.Endpoints[1] != want[1] || !req.AllowLAN {
		t.Fatalf("req %+v", req)
	}
	if r := parseKillSwitchSpec("", false, exists); len(r.Interfaces)+len(r.Endpoints) != 0 {
		t.Fatalf("empty spec: %+v", r)
	}
}

func TestHelperMissing(t *testing.T) {
	old := opsFunc
	defer func() { opsFunc = old }()
	opsFunc = func(context.Context) (privops.Ops, error) { return nil, privops.ErrHelperNotInstalled }

	ks := NewKillSwitch()
	if err := ks.Enable("wg0|1.2.3.4", false); !errors.Is(err, privops.ErrHelperNotInstalled) || ks.IsEnabled() {
		t.Fatalf("enable: %v enabled=%v", err, ks.IsEnabled())
	}
	if err := ks.Disable(); err != nil {
		t.Fatalf("disable when never enabled must not fail: %v", err)
	}
	g := NewDNSGuard()
	if err := g.Apply("wg0", []net.IP{net.ParseIP("10.0.0.53")}); !errors.Is(err, privops.ErrHelperNotInstalled) || g.IsActive() {
		t.Fatalf("apply: %v", err)
	}
}
