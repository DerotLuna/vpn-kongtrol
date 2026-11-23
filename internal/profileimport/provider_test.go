package profileimport

import (
	"net/netip"
	"slices"
	"testing"
)

// A Proton WireGuard file as Proton's download page writes it (comments
// under [Interface], 10.2.0.2 / 10.2.0.1; OPNsense how-to, accessed
// 2026-11-23), with the IPv6 lines added in 2026.
const protonConf = `[Interface]
# Bouncing = 0
# NetShield = 1
# Moderate NAT = off
# NAT-PMP (Port Forwarding) = off
# VPN Accelerator = on
PrivateKey = cGxhY2Vob2xkZXJwbGFjZWhvbGRlcnBsYWNlaG9sZGVyMDA=
Address = 10.2.0.2/32, 2a07:b944::2:2/128
DNS = 10.2.0.1, 2a07:b944::2:1

[Peer]
# NO#21
PublicKey = cGVlcnBlZXJwZWVycGVlcnBlZXJwZWVycGVlcnBlZXIwMDA=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.7:51820
`

func TestProviderForContent(t *testing.T) {
	for _, c := range []struct {
		name, typ, body, want string
	}{
		{"proton wireguard", "wireguard", protonConf, ProviderProton},
		{"proton second tunnel 10.3.0.2", "wireguard", "[Interface]\n#NetShield=0\n# VPN Accelerator = on\nAddress = 10.3.0.2/32\n", ProviderProton},
		{"proton dns only", "wireguard", "[Interface]\n# Bouncing = 3\n# Moderate NAT = on\nAddress = 172.16.0.2/32\nDNS = 10.2.0.1\n", ProviderProton},
		{"one comment key is not enough", "wireguard", "[Interface]\n# NetShield = 1\nAddress = 10.2.0.2/32\n", ""},
		{"comment keys but other addressing", "wireguard", "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\nAddress = 192.168.5.2/24\nDNS = 1.1.1.1\n", ""},
		{"plain wireguard", "wireguard", "[Interface]\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n", ""},
		{"proton ovpn verify-x509-name", "openvpn", "client\nremote 203.0.113.9 1194\nverify-x509-name \"us-ny-291.protonvpn.net\" name\n", ProviderProton},
		{"proton ovpn remote host", "openvpn", "client\nremote node-ch-02.protonvpn.net 1194\n", ProviderProton},
		{"lookalike host", "openvpn", "client\nremote evilprotonvpn.net.example 1194\nverify-x509-name server name\n", ""},
		{"commented proton host", "openvpn", "client\n# remote node-ch-02.protonvpn.net 1194\nremote vpn.example 1194\n", ""},
		{"wrong type", "openvpn", protonConf, ""},
	} {
		if got := ProviderForContent(c.typ, []byte(c.body)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestWireGuardAddresses(t *testing.T) {
	got := WireGuardAddresses([]byte(protonConf + "\n[Interface]\nAddress = 10.3.0.2\n"))
	want := []netip.Addr{netip.MustParseAddr("10.2.0.2"), netip.MustParseAddr("2a07:b944::2:2"), netip.MustParseAddr("10.3.0.2")}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// Address under [Peer] is not a tunnel address.
	if got := WireGuardAddresses([]byte("[Peer]\nAddress = 10.2.0.2/32\n")); len(got) != 0 {
		t.Fatalf("peer address taken: %v", got)
	}
}
