package main

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// countingAdapter tracks how many times Status/TunnelInfo were called, so
// tests can prove filteredRows stopped hitting the adapter directly once a
// Collector snapshot is available.
type countingAdapter struct {
	status     vpn.Status
	info       *vpn.TunnelInfo
	statusHits int32
	infoHits   int32
}

func (a *countingAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (a *countingAdapter) Disconnect(context.Context) error                 { return nil }
func (a *countingAdapter) Reconnect(context.Context) error                  { return nil }
func (a *countingAdapter) Name() string                                     { return "mock" }
func (a *countingAdapter) Version() string                                  { return "v0" }
func (a *countingAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }
func (a *countingAdapter) Status() vpn.Status {
	atomic.AddInt32(&a.statusHits, 1)
	return a.status
}
func (a *countingAdapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	atomic.AddInt32(&a.infoHits, 1)
	return a.info, nil
}

func TestFilteredRows_FallsBackToAdaptersWithoutCollector(t *testing.T) {
	ad := &countingAdapter{
		status: vpn.StatusConnected,
		info: &vpn.TunnelInfo{
			AssignedIP:  net.ParseIP("10.0.0.5"),
			ConnectedAt: time.Now().Add(-90 * time.Second),
		},
	}
	m := upModel{
		adapters: map[string]vpn.VPNAdapter{"office": ad},
		now:      time.Now(),
		filter:   upFilterAll,
	}

	rows := m.filteredRows()
	if len(rows) != 1 {
		t.Fatalf("len(rows)=%d, want 1", len(rows))
	}
	if rows[0].IP != "10.0.0.5" {
		t.Fatalf("IP=%q, want 10.0.0.5", rows[0].IP)
	}
	if atomic.LoadInt32(&ad.statusHits) == 0 || atomic.LoadInt32(&ad.infoHits) == 0 {
		t.Fatal("expected direct adapter calls in the no-collector fallback path")
	}
}

func TestFilteredRows_UsesCollectorSnapshotWhenAvailable(t *testing.T) {
	ad := &countingAdapter{
		status: vpn.StatusConnected,
		info: &vpn.TunnelInfo{
			AssignedIP:  net.ParseIP("10.0.0.5"),
			ConnectedAt: time.Now().Add(-90 * time.Second),
		},
	}
	col := monitor.NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	col.Start(time.Hour) // primes one collect() synchronously, then goes quiet
	defer col.Stop()

	hitsAfterPrime := atomic.LoadInt32(&ad.statusHits)
	if hitsAfterPrime == 0 {
		t.Fatal("expected the collector's own priming collect() to have hit the adapter")
	}

	m := upModel{
		adapters: map[string]vpn.VPNAdapter{"office": ad},
		now:      time.Now(),
		filter:   upFilterAll,
		col:      col,
	}

	rows := m.filteredRows()
	if len(rows) != 1 {
		t.Fatalf("len(rows)=%d, want 1", len(rows))
	}
	if rows[0].Status != vpn.StatusConnected {
		t.Fatalf("Status=%v, want connected", rows[0].Status)
	}
	if rows[0].IP != "10.0.0.5" {
		t.Fatalf("IP=%q, want 10.0.0.5", rows[0].IP)
	}
	// The render path itself must not have called into the adapter again —
	// it should have read purely from the collector's cached snapshot.
	if got := atomic.LoadInt32(&ad.statusHits); got != hitsAfterPrime {
		t.Fatalf("filteredRows called adapter.Status() directly (hits %d -> %d); expected it to read from the collector snapshot instead", hitsAfterPrime, got)
	}
}

func TestFilteredRows_UsesRemoteSnapshotWhenDaemonFound(t *testing.T) {
	ad := &countingAdapter{status: vpn.StatusDisconnected} // local instance, never actually connected
	m := upModel{
		adapters:        map[string]vpn.VPNAdapter{"office": ad},
		now:             time.Now(),
		filter:          upFilterAll,
		remoteConnected: true,
		remoteSnapshot: map[string]monitor.TunnelMetrics{
			"office": {
				Name:        "office",
				Status:      vpn.StatusConnected,
				AssignedIP:  "10.9.9.9",
				ConnectedAt: time.Now().Add(-time.Minute),
			},
		},
	}

	rows := m.filteredRows()
	if len(rows) != 1 {
		t.Fatalf("len(rows)=%d, want 1", len(rows))
	}
	if rows[0].Status != vpn.StatusConnected {
		t.Fatalf("Status=%v, want connected (from the real daemon's remote snapshot, not this process's own disconnected local adapter)", rows[0].Status)
	}
	if rows[0].IP != "10.9.9.9" {
		t.Fatalf("IP=%q, want 10.9.9.9", rows[0].IP)
	}

	// Without a live remote connection, must fall back to the local adapter
	// (disconnected) instead of showing stale remote data.
	m.remoteConnected = false
	rows = m.filteredRows()
	if len(rows) != 1 || rows[0].Status != vpn.StatusDisconnected {
		t.Fatalf("expected local fallback (disconnected) once remoteConnected=false, got %+v", rows)
	}
}

func TestDaemonWSURL(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:9741":  "ws://127.0.0.1:9741/api/v1/ws/metrics",
		"https://127.0.0.1:9741": "wss://127.0.0.1:9741/api/v1/ws/metrics",
	}
	for in, want := range cases {
		if got := daemonWSURL(in); got != want {
			t.Errorf("daemonWSURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUpModel_PassesFilter(t *testing.T) {
	cases := []struct {
		filter upFilter
		status vpn.Status
		want   bool
	}{
		{upFilterAll, vpn.StatusDisconnected, true},
		{upFilterConnected, vpn.StatusConnected, true},
		{upFilterConnected, vpn.StatusDisconnected, false},
		{upFilterConnecting, vpn.StatusConnecting, true},
		{upFilterError, vpn.StatusError, true},
		{upFilterDisconnected, vpn.StatusConnected, false},
		{upFilterDisconnected, vpn.StatusDisconnected, true},
	}
	for _, c := range cases {
		m := upModel{filter: c.filter}
		if got := m.passesFilter(c.status); got != c.want {
			t.Errorf("passesFilter(filter=%v, status=%v) = %v, want %v", c.filter, c.status, got, c.want)
		}
	}
}
