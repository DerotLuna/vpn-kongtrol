package openvpn

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigRoutes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.ovpn")
	body := "client\n" +
		"route 10.20.0.0 255.255.0.0\n" +
		"route 10.20.0.0 255.255.0.0 vpn_gateway 5\n" + // duplicate
		"route 8.8.8.8\n" + // no mask: /32
		"route 192.168.1.0 255.255.255.0 net_gateway\n" + // off the tunnel
		"route intranet.example 255.255.255.255\n" + // hostname
		"route 10.0.0.0 255.0.255.0\n" + // non-contiguous mask
		"--route-ipv6 2001:db8::/32\n" +
		"# route 1.1.1.1\n; route 2.2.2.2\n" +
		"redirect-gateway def1 ipv6\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ConfigRoutes(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.20.0.0/16", "8.8.8.8/32", "2001:db8::/32", "::/0", "0.0.0.0/0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, err := ConfigRoutes(filepath.Join(t.TempDir(), "missing.ovpn")); err == nil {
		t.Error("a missing file must return its error")
	}
}
