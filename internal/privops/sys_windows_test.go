//go:build windows

package privops

import (
	"reflect"
	"testing"
)

// TestDNSListArgs pins the netsh argv to the documented named-parameter
// syntax (learn.microsoft.com, "netsh interface": ipv4|ipv6 set/add
// dnsservers).
func TestDNSListArgs(t *testing.T) {
	got := dnsListArgs("ipv4", "Ethernet 2", []string{"10.0.0.53", "10.0.1.53", "1.1.1.1"})
	want := [][]string{
		{"interface", "ipv4", "set", "dnsservers", "name=Ethernet 2", "source=static", "address=10.0.0.53", "register=primary", "validate=no"},
		{"interface", "ipv4", "add", "dnsservers", "name=Ethernet 2", "address=10.0.1.53", "index=2", "validate=no"},
		{"interface", "ipv4", "add", "dnsservers", "name=Ethernet 2", "address=1.1.1.1", "index=3", "validate=no"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("static:\n got %q\nwant %q", got, want)
	}
	got = dnsListArgs("ipv4", "Wi-Fi", nil)
	want = [][]string{{"interface", "ipv4", "set", "dnsservers", "name=Wi-Fi", "source=dhcp"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dhcp:\n got %q\nwant %q", got, want)
	}
	got = dnsListArgs("ipv6", "wg0", []string{"fd00::53"})
	if got[0][1] != "ipv6" || got[0][6] != "address=fd00::53" {
		t.Fatalf("ipv6: %q", got)
	}
}

func TestServiceExe(t *testing.T) {
	for in, want := range map[string]string{
		`"C:\Program Files\WireGuard\wireguard.exe" /tunnelservice "C:\x\rluna.conf"`: `C:\Program Files\WireGuard\wireguard.exe`,
		`C:\Program Files\WireGuard\wireguard.exe /tunnelservice x.conf`:              `C:\Program Files\WireGuard\wireguard.exe`,
		`C:\Windows\System32\svchost.exe -k netsvcs`:                                  `C:\Windows\System32\svchost.exe`,
		``: ``,
	} {
		if got := serviceExe(in); got != want {
			t.Errorf("serviceExe(%q) = %q, want %q", in, got, want)
		}
	}
}
