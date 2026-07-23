package policy

import (
	"fmt"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func BenchmarkEngineResolveApp(b *testing.B) {
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{"work": {Type: "wireguard"}},
	}
	for i := range 100 {
		cfg.Policies = append(cfg.Policies, config.PolicyRule{
			Name: fmt.Sprintf("rule-%d", i),
			Via:  "work",
			Match: config.MatchSpec{
				Apps: []string{
					fmt.Sprintf("tool-%d", i),
					fmt.Sprintf("helper-%d*", i),
					fmt.Sprintf("client-%d.exe", i),
				},
			},
		})
	}
	engine, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		engine.ResolveApp(`C:\Program Files\Kongtrol\client-99.exe`)
	}
}
