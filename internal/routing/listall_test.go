package routing

import "testing"

func routeStrings(rs []Route) []string {
	var out []string
	for _, r := range rs {
		gw := "-"
		if r.Gateway != nil {
			gw = r.Gateway.String()
		}
		out = append(out, r.Destination.String()+" "+gw+" "+r.Interface)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestParseNetRouteLines(t *testing.T) {
	out := "10.200.0.0/24|192.168.220.249|Local Area Connection|0\r\n0.0.0.0/0|192.168.2.1|Wi-Fi|0\r\n192.168.220.248/30|0.0.0.0|Local Area Connection|256\r\nff00::/8|::|rluna|256\r\n"
	eq(t, routeStrings(parseNetRouteLines(out)), []string{
		"10.200.0.0/24 192.168.220.249 Local Area Connection",
		"0.0.0.0/0 192.168.2.1 Wi-Fi",
		"192.168.220.248/30 - Local Area Connection",
		"ff00::/8 - rluna",
	})
}

func TestParseIPRouteLines(t *testing.T) {
	out := "default via 192.168.1.1 dev wlan0 proto dhcp metric 600\n10.200.0.0/24 via 10.8.0.1 dev tun0\n10.8.0.1 dev tun0 proto kernel scope link src 10.8.0.2\nunreachable 10.99.0.0/16\n"
	eq(t, routeStrings(parseIPRouteLines(out, false)), []string{
		"0.0.0.0/0 192.168.1.1 wlan0",
		"10.200.0.0/24 10.8.0.1 tun0",
		"10.8.0.1/32 - tun0",
	})
}

func TestParseNetstatRoutes(t *testing.T) {
	out := `Routing tables

Internet:
Destination        Gateway            Flags               Netif Expire
default            192.168.1.1        UGScg                 en0
10.200             10.8.0.1           UGSc                utun4
10.8.0.1           10.8.0.2           UH                  utun4
10.200.7/24        10.8.0.1           UGSc                utun4

Internet6:
Destination                             Gateway                                 Flags               Netif Expire
fe80::%utun4/64                         fe80::1%utun4                           UcI                 utun4
`
	eq(t, routeStrings(parseNetstatRoutes(out)), []string{
		"0.0.0.0/0 192.168.1.1 en0",
		"10.200.0.0/16 10.8.0.1 utun4",
		"10.8.0.1/32 10.8.0.2 utun4",
		"10.200.7.0/24 10.8.0.1 utun4",
		"fe80::/64 - utun4",
	})
}
