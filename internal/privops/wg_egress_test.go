package privops

import (
	"strings"
	"testing"
)

const dualStackConfig = `[Interface]
PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
Address = 10.7.0.2/32, fd43:1::2/128
DNS = 10.7.0.1, vpn.example # optional search domain

[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
Endpoint = vpn.example:51820
AllowedIPs = 0.0.0.0/0, ::/0
`

func TestInspectWireGuardDefaultEgress(t *testing.T) {
	tests := []struct {
		name, old, replacement, reason string
	}{
		{name: "valid"},
		{name: "IPv6 endpoint", old: "vpn.example:51820", replacement: "[2001:db8::123]:51820"},
		{name: "automatic table", old: "[Peer]", replacement: "Table = auto\n[Peer]"},
		{name: "explicit table", old: "[Peer]", replacement: "Table = 51829\n[Peer]", reason: "unsupported_options"},
		{name: "routes off", old: "[Peer]", replacement: "Table = off\n[Peer]", reason: "unsupported_options"},
		{name: "configured mark", old: "[Peer]", replacement: "FwMark = 7\n[Peer]", reason: "unsupported_options"},
		{name: "save overrides file", old: "[Peer]", replacement: "SaveConfig = true\n[Peer]", reason: "unsupported_options"},
		{name: "shell hook", old: "[Peer]", replacement: "PostUp = true\n[Peer]", reason: "invalid_config"},
		{name: "split prefixes", old: "0.0.0.0/0", replacement: "0.0.0.0/1, 128.0.0.0/1", reason: "missing_defaults"},
		{name: "no IPv6 default", old: ", ::/0", reason: "missing_defaults"},
		{name: "no IPv4 default", old: "0.0.0.0/0, ", reason: "missing_defaults"},
		{name: "IPv6 link local", old: "fd43:1::2/128", replacement: "fe80::1/128", reason: "missing_addresses"},
		{name: "DNS search domain only", old: "10.7.0.1, ", reason: "missing_dns"},
		{name: "missing endpoint port", old: ":51820", reason: "missing_endpoint"},
		{name: "invalid peer key", old: "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=", replacement: "invalid", reason: "invalid_config"},
		{name: "multiple peers", old: ", ::/0", replacement: ", ::/0\n[Peer]\nPublicKey = invalid", reason: "peer_count"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := dualStackConfig
			if tt.old != "" {
				content = strings.Replace(content, tt.old, tt.replacement, 1)
			}
			want := DefaultEgressCheck{Compatible: tt.reason == "", Reason: tt.reason}
			if got := inspectWireGuardDefaultEgress([]byte(content)); got != want {
				t.Errorf("check = %+v, want %+v", got, want)
			}
		})
	}
}
