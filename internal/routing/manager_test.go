package routing

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestToPrivRoute(t *testing.T) {
	_, n, _ := net.ParseCIDR("10.20.0.0/16")
	pr, err := toPrivRoute(Route{Destination: *n, Gateway: net.ParseIP("10.8.0.1"), Interface: "tun0", Metric: 5})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Destination.String() != "10.20.0.0/16" || pr.Gateway.String() != "10.8.0.1" || pr.Interface != "tun0" || pr.Metric != 5 {
		t.Fatalf("got %+v", pr)
	}
	// 16-byte IPv4 with a 16-byte mask (as built by some callers).
	n16 := net.IPNet{IP: net.ParseIP("192.168.5.0"), Mask: net.CIDRMask(120, 128)}
	if pr, err = toPrivRoute(Route{Destination: n16, Interface: "wg0"}); err != nil || pr.Destination.String() != "192.168.5.0/24" {
		t.Fatalf("v4-in-v6: %+v %v", pr, err)
	}
}

func TestAdd_HelperMissingKeepsSentinel(t *testing.T) {
	old := opsFunc
	defer func() { opsFunc = old }()
	opsFunc = func(context.Context) (privops.Ops, error) { return nil, privops.ErrHelperNotInstalled }
	_, n, _ := net.ParseCIDR("10.0.0.0/8")
	m := NewRouteManager()
	err := m.Add(Route{Destination: *n, Interface: "tun0"})
	if !errors.Is(err, privops.ErrHelperNotInstalled) {
		t.Fatalf("err = %v, want ErrHelperNotInstalled", err)
	}
	if rs, _ := m.(*manager).routes, 0; len(rs) != 0 {
		t.Fatal("failed route was tracked")
	}
}
