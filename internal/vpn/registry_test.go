package vpn_test

import (
	"context"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// mockAdapter is a minimal VPNAdapter for testing the registry.
type mockAdapter struct{ name string }

func (m *mockAdapter) Connect(_ context.Context, _ vpn.AdapterConfig) error { return nil }
func (m *mockAdapter) Disconnect(_ context.Context) error                   { return nil }
func (m *mockAdapter) Reconnect(_ context.Context) error                    { return nil }
func (m *mockAdapter) Status() vpn.Status                                   { return vpn.StatusDisconnected }
func (m *mockAdapter) TunnelInfo() (*vpn.TunnelInfo, error)                 { return nil, nil }
func (m *mockAdapter) Name() string                                          { return m.name }
func (m *mockAdapter) Version() string                                       { return "1.0" }
func (m *mockAdapter) Capabilities() vpn.Capabilities                       { return vpn.Capabilities{} }

func TestRegistry_RegisterAndNew(t *testing.T) {
	vpn.Register("mock-test-adapter", func() vpn.VPNAdapter {
		return &mockAdapter{name: "mock-test-adapter"}
	})

	a, err := vpn.New("mock-test-adapter")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if a.Name() != "mock-test-adapter" {
		t.Errorf("Name = %q, want mock-test-adapter", a.Name())
	}
}

func TestRegistry_UnknownAdapter(t *testing.T) {
	_, err := vpn.New("does-not-exist-xyz")
	if err == nil {
		t.Error("expected error for unknown adapter, got nil")
	}
}

func TestRegistry_NewCreatesNewInstance(t *testing.T) {
	vpn.Register("mock-instance-test", func() vpn.VPNAdapter {
		return &mockAdapter{name: "mock-instance-test"}
	})

	a1, _ := vpn.New("mock-instance-test")
	a2, _ := vpn.New("mock-instance-test")

	if a1 == a2 {
		t.Error("New should return different instances each call")
	}
}

func TestRegistry_Registered_ContainsAdapter(t *testing.T) {
	vpn.Register("mock-listed", func() vpn.VPNAdapter {
		return &mockAdapter{name: "mock-listed"}
	})

	names := vpn.Registered()
	found := false
	for _, n := range names {
		if n == "mock-listed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Registered() did not include mock-listed")
	}
}
