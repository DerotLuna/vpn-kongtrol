package monitor

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// BenchmarkPolicyResolverMatchAddr measures the matcher used by the TCP
// connection monitor, excluding the OS table scan and DNS lookups.
func BenchmarkPolicyResolverMatchAddr(b *testing.B) {
	destinations := [...]netip.Addr{
		netip.MustParseAddr("10.0.0.42"),
		netip.MustParseAddr("10.0.127.42"),
		netip.MustParseAddr("2001:db8::42"),
		netip.MustParseAddr("192.0.2.42"),
	}
	for _, size := range []int{8, 128} {
		b.Run(fmt.Sprintf("rules=%d", size), func(b *testing.B) {
			cfg := &config.Config{Policies: make([]config.PolicyRule, 0, size)}
			for i := range size {
				cfg.Policies = append(cfg.Policies, config.PolicyRule{
					Name: fmt.Sprintf("network-%d", i),
					Via:  "work",
					Match: config.MatchSpec{IPRanges: []string{
						fmt.Sprintf("10.0.%d.0/24", i),
					}},
				})
			}
			resolver := NewPolicyResolver(cfg, nil, nil)
			b.ReportAllocs()
			for b.Loop() {
				for _, addr := range destinations {
					resolver.MatchAddr(addr)
				}
			}
		})
	}
}
