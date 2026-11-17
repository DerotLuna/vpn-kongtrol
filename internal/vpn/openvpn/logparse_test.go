package openvpn

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

const ts = "2026-11-17 10:00:00 "

func TestParseInterface(t *testing.T) {
	tests := []struct {
		name, line, want string
		ok               bool
	}{
		{"win dco", ts + "DCO device [OpenVPN Data Channel Offload] opened", "OpenVPN Data Channel Offload", true},
		{"win wintun", ts + "Wintun device [OpenVPN Wintun] opened", "OpenVPN Wintun", true},
		{"win tap", ts + `TAP-WIN32 device [OpenVPN TAP-Windows6] opened: \.\Global\{ABC}.tap`, "OpenVPN TAP-Windows6", true},
		{"linux tun", ts + "TUN/TAP device tun0 opened", "tun0", true},
		{"linux dco", ts + "DCO device tun0 opened", "tun0", true},
		{"macos utun", ts + "Opened utun device utun4", "utun4", true},
		{"negative", ts + "Initialization Sequence Completed", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseInterface(tt.line)
			if got != tt.want || ok != tt.ok {
				t.Errorf("parseInterface = %q,%v; want %q,%v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParsePushedDNS(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"classic", ts + "PUSH: Received control message: 'PUSH_REPLY,route 10.0.0.0 255.255.0.0,dhcp-option DNS 10.0.0.2,dhcp-option DNS 10.0.0.3,topology subnet,ifconfig 10.8.0.6 255.255.255.0'",
			[]string{"10.0.0.2", "10.0.0.3"}},
		{"openvpn 2.6", ts + "PUSH: Received control message: 'PUSH_REPLY,dns server 0 address 10.0.0.53 10.0.0.54:53,topology subnet'",
			[]string{"10.0.0.53", "10.0.0.54"}},
		{"ipv6", ts + "PUSH: Received control message: 'PUSH_REPLY,dns server 0 address [fd00::1]:53'",
			[]string{"fd00::1"}},
		{"dedup", ts + "PUSH: Received control message: 'PUSH_REPLY,dhcp-option DNS 10.0.0.2,dns server 0 address 10.0.0.2:53,dhcp-option DNS 10.0.0.2'",
			[]string{"10.0.0.2"}},
		{"non push", ts + "dhcp-option DNS 10.0.0.2", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePushedDNS(tt.line)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if !got[i].Equal(net.ParseIP(w)) {
					t.Errorf("[%d] = %v, want %s", i, got[i], w)
				}
			}
		})
	}
}

func TestLogState_TailCapped(t *testing.T) {
	var s logState
	for i := 0; i < maxTail+10; i++ {
		s.consume(fmt.Sprintf("line %d", i))
	}
	if len(s.tail) != maxTail {
		t.Fatalf("tail len = %d, want %d", len(s.tail), maxTail)
	}
	if s.tail[0] != fmt.Sprintf("line %d", 10) {
		t.Errorf("oldest = %q", s.tail[0])
	}
}

func TestLogState_LastErrors(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"prefers error lines", []string{
			ts + "OpenVPN 2.6.12 starting",
			ts + "Cannot load CA certificate file ca.crt  ",
			ts + "noise",
			ts + "Exiting due to fatal error",
			ts + "more noise",
		}, ts + "Cannot load CA certificate file ca.crt | " + ts + "Exiting due to fatal error"},
		{"fallback last 3", []string{"a", "b", "c", "d"}, "b | c | d"},
		{"fallback short", []string{"only"}, "only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s logState
			for _, l := range tt.lines {
				s.consume(l)
			}
			got := s.lastErrors()
			if got != strings.TrimSpace(tt.want) {
				t.Errorf("lastErrors = %q, want %q", got, tt.want)
			}
		})
	}
}
