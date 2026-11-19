package privops

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestResolvConfManagedByResolved(t *testing.T) {
	for target, want := range map[string]bool{
		"/run/systemd/resolve/stub-resolv.conf": true,
		"/run/systemd/resolve/resolv.conf":      true,
		"/run/systemd/resolve/../x":             false,
		"/etc/resolv.conf":                      false,
		"/run/resolvconf/resolv.conf":           false,
		"/run/NetworkManager/resolv.conf":       false,
	} {
		if got := resolvConfManagedByResolved(target); got != want {
			t.Errorf("%s: got %v, want %v", target, got, want)
		}
	}
}

func TestResolvectlSetArgs(t *testing.T) {
	req := DNSRequest{Interface: "tun0", Servers: []netip.Addr{netip.MustParseAddr("10.8.0.1"), netip.MustParseAddr("fd00::1")}}
	want := [][]string{
		{"dns", "tun0", "10.8.0.1", "fd00::1"},
		{"domain", "tun0", "~."},
		{"default-route", "tun0", "yes"},
	}
	if got := resolvectlSetArgs(req); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
