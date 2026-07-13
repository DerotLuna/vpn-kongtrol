//go:build e2e

package vpn_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

type liveScenario struct {
	name       string
	adapter    string
	config     vpn.AdapterConfig
	requireEnv []string
}

func TestLiveDaemons_ConnectStatusReconnectDisconnect(t *testing.T) {
	if os.Getenv("KONGTROL_E2E") != "1" {
		t.Skip("set KONGTROL_E2E=1 to run live daemon E2E tests")
	}

	scenarios := []liveScenario{
		{
			name:    "openvpn",
			adapter: "openvpn",
			config: vpn.AdapterConfig{
				ConfigPath: os.Getenv("KONGTROL_E2E_OPENVPN_CONFIG"),
				CertPath:   os.Getenv("KONGTROL_E2E_OPENVPN_CERT"),
				KeyPath:    os.Getenv("KONGTROL_E2E_OPENVPN_KEY"),
				Username:   os.Getenv("KONGTROL_E2E_OPENVPN_USERNAME"),
				Password:   os.Getenv("KONGTROL_E2E_OPENVPN_PASSWORD"),
			},
			requireEnv: []string{"KONGTROL_E2E_OPENVPN_CONFIG"},
		},
		{
			name:    "wireguard",
			adapter: "wireguard",
			config: vpn.AdapterConfig{
				ConfigPath: os.Getenv("KONGTROL_E2E_WIREGUARD_CONFIG"),
				TunnelName: os.Getenv("KONGTROL_E2E_WIREGUARD_IFACE"),
			},
			requireEnv: []string{"KONGTROL_E2E_WIREGUARD_CONFIG"},
		},
	}

	for _, tc := range scenarios {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range tc.requireEnv {
				if os.Getenv(key) == "" {
					t.Skipf("missing required env: %s", key)
				}
			}

			a, err := vpn.New(tc.adapter)
			if err != nil {
				t.Fatalf("vpn.New(%q): %v", tc.adapter, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if err := a.Connect(ctx, tc.config); err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_ = a.Disconnect(cleanupCtx)
			}()

			if got := a.Status(); got != vpn.StatusConnected {
				t.Fatalf("Status after connect = %q, want %q", got, vpn.StatusConnected)
			}

			if _, err := a.TunnelInfo(); err != nil {
				t.Fatalf("TunnelInfo: %v", err)
			}

			if err := a.Reconnect(ctx); err != nil {
				t.Fatalf("Reconnect: %v", err)
			}

			if got := a.Status(); got != vpn.StatusConnected {
				t.Fatalf("Status after reconnect = %q, want %q", got, vpn.StatusConnected)
			}

			if err := a.Disconnect(ctx); err != nil {
				t.Fatalf("Disconnect: %v", err)
			}

			if got := a.Status(); got != vpn.StatusDisconnected {
				t.Fatalf("Status after disconnect = %q, want %q", got, vpn.StatusDisconnected)
			}
		})
	}
}
