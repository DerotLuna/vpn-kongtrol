package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type recordingWireGuardAdapter struct {
	blockingAdapter
	config vpn.AdapterConfig
	data   string
}

func (a *recordingWireGuardAdapter) Connect(_ context.Context, cfg vpn.AdapterConfig) error {
	a.config = cfg
	data, err := os.ReadFile(cfg.ConfigPath)
	a.data = string(data)
	return err
}

func TestConnectKeepsWireGuardDefaultEgressConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.conf")
	const conf = `[Interface]
PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
Address = 10.7.0.2/32, fd43:1::2/128
DNS = 10.7.0.1
[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
Endpoint = 192.0.2.2:51820
AllowedIPs = 0.0.0.0/0, ::/0
`
	if err := os.WriteFile(path, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingWireGuardAdapter{}
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{"native": {Type: "wireguard", ConfigFile: path, DefaultEgress: true}},
		Policies: []config.PolicyRule{{
			Name: "specific", Via: "native", Match: config.MatchSpec{IPRanges: []string{"198.51.100.12/32"}},
		}},
	}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:      cfg,
		Adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"native": adapter}),
	})
	if err := svc.Connect(context.Background(), "native"); err != nil {
		t.Fatal(err)
	}
	if adapter.config.ConfigPath != path || adapter.data != conf || !adapter.config.DefaultEgress {
		t.Fatal("selected WireGuard profile did not reach the adapter with native defaults")
	}

	if err := os.WriteFile(path, []byte(strings.Replace(conf, "0.0.0.0/0, ::/0", "10.7.0.0/24", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.Connect(context.Background(), "native"); err == nil {
		t.Fatal("connected with the missing defaults")
	}
	cfg.VPNs["native"] = config.VPNConfig{Type: "wireguard", ConfigFile: path}
	if err := svc.Connect(context.Background(), "native"); err != nil {
		t.Fatal(err)
	}
	if adapter.config.ConfigPath == path || adapter.config.DefaultEgress || !strings.Contains(adapter.data, "198.51.100.12/32") {
		t.Fatal("split-tunnel policy did not get its usual patched configuration")
	}
}
