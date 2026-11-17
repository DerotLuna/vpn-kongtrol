package monitor

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type collectorAdapter struct {
	status vpn.Status
	info   *vpn.TunnelInfo
}

func TestCollector_LoadSaveHistory(t *testing.T) {
	ad := &collectorAdapter{status: vpn.StatusConnected}
	c := NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	c.RecordReconnect("office")
	c.RecordLeak("office")
	path := filepath.Join(t.TempDir(), "history.json")

	if err := c.SaveHistory(path); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}

	c2 := NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	if err := c2.LoadHistory(path); err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	h := c2.HistorySnapshot()["office"]
	if h.Reconnects != 1 || h.Leaks != 1 {
		t.Fatalf("loaded history mismatch: %+v", h)
	}
}

func (a *collectorAdapter) Name() string                                         { return "mock" }
func (a *collectorAdapter) Version() string                                      { return "0.0.0" }
func (a *collectorAdapter) Capabilities() vpn.Capabilities                       { return vpn.Capabilities{} }
func (a *collectorAdapter) Connect(_ context.Context, _ vpn.AdapterConfig) error { return nil }
func (a *collectorAdapter) Disconnect(_ context.Context) error                   { return nil }
func (a *collectorAdapter) Reconnect(_ context.Context) error                    { return nil }
func (a *collectorAdapter) Status() vpn.Status                                   { return a.status }
func (a *collectorAdapter) TunnelInfo() (*vpn.TunnelInfo, error)                 { return a.info, nil }

func TestCollector_HistoryAndHealth(t *testing.T) {
	ad := &collectorAdapter{
		status: vpn.StatusConnected,
		info: &vpn.TunnelInfo{
			InterfaceName: "wg0",
			AssignedIP:    net.ParseIP("10.2.0.10"),
			ConnectedAt:   time.Now().Add(-10 * time.Minute),
		},
	}
	c := NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	c.collect()
	c.RecordReconnect("office")
	c.RecordLeak("office")
	c.RecordHealth("office", 120*time.Millisecond, true, true)

	h := c.HistorySnapshot()["office"]
	if h.Reconnects != 1 {
		t.Fatalf("Reconnects=%d, want 1", h.Reconnects)
	}
	if h.Leaks != 1 {
		t.Fatalf("Leaks=%d, want 1", h.Leaks)
	}
	if h.LastLatencyMS != 120 {
		t.Fatalf("LastLatencyMS=%d, want 120", h.LastLatencyMS)
	}
	if !h.LastReachable || !h.LastDNSOK {
		t.Fatalf("health flags not recorded: reachable=%v dns=%v", h.LastReachable, h.LastDNSOK)
	}
	if h.LastConnectedAt.IsZero() {
		t.Fatal("LastConnectedAt should be recorded for connected profile")
	}
}

func TestCollector_TracksLastConnectedAndDown(t *testing.T) {
	ad := &collectorAdapter{
		status: vpn.StatusDisconnected,
		info:   &vpn.TunnelInfo{},
	}
	c := NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	c.collect()

	ad.status = vpn.StatusConnected
	ad.info = &vpn.TunnelInfo{
		ConnectedAt: time.Now().Add(-2 * time.Minute),
	}
	c.collect()

	mid := c.HistorySnapshot()["office"]
	if mid.LastConnectedAt.IsZero() {
		t.Fatal("LastConnectedAt should be set after transition to connected")
	}
	if !mid.LastDownAt.IsZero() {
		t.Fatal("LastDownAt should be empty while connected")
	}

	ad.status = vpn.StatusDisconnected
	ad.info = &vpn.TunnelInfo{}
	c.collect()

	final := c.HistorySnapshot()["office"]
	if final.LastDownAt.IsZero() {
		t.Fatal("LastDownAt should be set after transition to disconnected")
	}
}

func TestCollector_SubscribeNotifiesOnlyOnRealChange(t *testing.T) {
	ad := &collectorAdapter{status: vpn.StatusDisconnected, info: &vpn.TunnelInfo{}}
	c := NewCollector(map[string]vpn.VPNAdapter{"office": ad})

	ch, cancel := c.Subscribe()
	defer cancel()

	c.collect() // first pass always transitions from "no prior state" — expect a signal
	select {
	case <-ch:
	default:
		t.Fatal("expected a change notification on the first collect")
	}

	c.collect() // nothing changed since the last pass — expect no signal
	select {
	case <-ch:
		t.Fatal("did not expect a change notification when nothing changed")
	default:
	}

	ad.status = vpn.StatusConnected
	ad.info = &vpn.TunnelInfo{ConnectedAt: time.Now()}
	c.collect() // status actually changed — expect a signal
	select {
	case <-ch:
	default:
		t.Fatal("expected a change notification after a status transition")
	}

	cancel()
	ad.status = vpn.StatusDisconnected
	c.collect() // unsubscribed — must not panic or block on a closed/removed subscriber
}

func TestCollector_SubscribeSendIsNonBlocking(t *testing.T) {
	ad := &collectorAdapter{status: vpn.StatusDisconnected, info: &vpn.TunnelInfo{}}
	c := NewCollector(map[string]vpn.VPNAdapter{"office": ad})

	ch, cancel := c.Subscribe()
	defer cancel()

	// Never drain ch. Multiple changes must coalesce instead of blocking collect().
	for i := 0; i < 3; i++ {
		if ad.status == vpn.StatusConnected {
			ad.status = vpn.StatusDisconnected
		} else {
			ad.status = vpn.StatusConnected
		}
		c.collect()
	}

	select {
	case <-ch:
	default:
		t.Fatal("expected at least one coalesced notification")
	}
}

func TestCollector_FallsBackToInterfaceCounters(t *testing.T) {
	orig := interfaceCounters
	t.Cleanup(func() { interfaceCounters = orig })
	interfaceCounters = func(name string) (uint64, uint64, bool) {
		if name != "wg-test" {
			return 0, 0, false
		}
		return 111, 222, true
	}
	ad := &collectorAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "wg-test"}}
	c := NewCollector(map[string]vpn.VPNAdapter{"p": ad})
	c.collect()
	m := c.Snapshot()["p"]
	if m.BytesSent != 111 || m.BytesReceived != 222 {
		t.Fatalf("fallback counters not used: %+v", m)
	}

	// Adapter-reported counters win over the OS fallback.
	ad.info = &vpn.TunnelInfo{InterfaceName: "wg-test", BytesSent: 5, BytesReceived: 6}
	c.collect()
	m = c.Snapshot()["p"]
	if m.BytesSent != 5 || m.BytesReceived != 6 {
		t.Fatalf("adapter counters overridden: %+v", m)
	}
}

func TestCollector_KeepsLatencyBetweenProbes(t *testing.T) {
	ad := &collectorAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{}}
	c := NewCollector(map[string]vpn.VPNAdapter{"p": ad})
	c.collect()
	c.RecordHealth("p", 42*time.Millisecond, true, true)
	c.collect() // a later 5s pass must not wipe the 30s probe's reading
	if got := c.Snapshot()["p"].LatencyMS; got != 42 {
		t.Fatalf("LatencyMS = %d, want 42", got)
	}
	ad.status = vpn.StatusDisconnected
	c.collect()
	if got := c.Snapshot()["p"].LatencyMS; got != 0 {
		t.Fatalf("LatencyMS after disconnect = %d, want 0", got)
	}
}
