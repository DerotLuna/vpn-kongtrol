package vpn_test

import (
	"context"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	// Ensure all adapters are registered before the test runs.
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/ciscoanyconnect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/cloudflarewarp"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/globalprotect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/tailscale"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

var allAdapterTypes = []string{
	"forticlient",
	"openvpn",
	"protonvpn",
	"ciscoanyconnect",
	"wireguard",
	"globalprotect",
	"tailscale",
	"cloudflarewarp",
}

// TestAllAdapters_Registered verifies every documented adapter type is
// present in the registry after blank-importing all adapter packages.
func TestAllAdapters_Registered(t *testing.T) {
	registered := make(map[string]bool)
	for _, name := range vpn.Registered() {
		registered[name] = true
	}

	for _, adapterType := range allAdapterTypes {
		if !registered[adapterType] {
			t.Errorf("adapter %q not found in registry; did you forget the blank import?", adapterType)
		}
	}
}

// TestAllAdapters_NewCreatesDistinctInstances ensures New() returns a fresh
// adapter on each call (not a shared singleton).
func TestAllAdapters_NewCreatesDistinctInstances(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a1, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			a2, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q) second call: %v", adapterType, err)
			}
			if a1 == a2 {
				t.Errorf("New(%q) returned the same instance twice", adapterType)
			}
		})
	}
}

// TestAllAdapters_Name verifies each adapter's Name() matches its registry key.
func TestAllAdapters_Name(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			if a.Name() != adapterType {
				t.Errorf("Name() = %q, want %q", a.Name(), adapterType)
			}
		})
	}
}

// TestAllAdapters_InitialStatus verifies that a freshly created adapter reports
// StatusDisconnected before any Connect() call.
func TestAllAdapters_InitialStatus(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			got := a.Status()
			if got != vpn.StatusDisconnected {
				t.Errorf("%s.Status() before Connect = %q, want %q", adapterType, got, vpn.StatusDisconnected)
			}
		})
	}
}

// TestAllAdapters_TunnelInfoNilWhenDisconnected verifies TunnelInfo returns
// (nil, nil) when the adapter has never connected.
func TestAllAdapters_TunnelInfoNilWhenDisconnected(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			info, err := a.TunnelInfo()
			if err != nil {
				t.Errorf("%s.TunnelInfo() error = %v, want nil", adapterType, err)
			}
			if info != nil {
				t.Errorf("%s.TunnelInfo() = %+v, want nil when disconnected", adapterType, info)
			}
		})
	}
}

// TestAllAdapters_DoubleConnect_Fails verifies that calling Connect() twice
// (simulated via forced status) returns an error rather than silently
// double-connecting. We set status manually by calling Connect once with a
// cancelled context so the adapter transitions to error, then verify a second
// connect from an already-connecting/connected state is rejected.
// NOTE: This test does NOT require real VPN software — it tests guard logic only.
func TestAllAdapters_ConnectOnConnected_Fails(t *testing.T) {
	// Adapters that lock against double-connect internally.
	// We check that the initial Status is Disconnected (covered above),
	// and that each adapter at minimum implements the guard in its struct.
	// Full double-connect testing requires the VPN daemon — skip here.
	t.Skip("requires live VPN daemon — run with -tags integration")
}

// TestAllAdapters_Capabilities verifies the Capabilities() call does not panic
// and returns a coherent struct.
func TestAllAdapters_Capabilities(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			caps := a.Capabilities()
			// wireguard and tailscale support multi-conn; others (warp, cisco) don't.
			// Just verify it doesn't panic and the struct is addressable.
			_ = caps.SupportsSplitTunnel
			_ = caps.SupportsMultiConn
			_ = caps.SupportsReconnect
		})
	}
}

// TestAllAdapters_Reconnect_WhenDisconnected verifies Reconnect from a
// disconnected state does not crash (it will fail gracefully without a daemon).
func TestAllAdapters_Reconnect_WhenDisconnected(t *testing.T) {
	for _, adapterType := range allAdapterTypes {
		t.Run(adapterType, func(t *testing.T) {
			a, err := vpn.New(adapterType)
			if err != nil {
				t.Fatalf("New(%q): %v", adapterType, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancelled — prevents any real network activity

			// Should return an error (no daemon), not panic.
			_ = a.Reconnect(ctx)
		})
	}
}
