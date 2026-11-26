package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestConnectAlreadyCancelledDoesNotStartAdapter(t *testing.T) {
	adapter := &blockingAdapter{started: make(chan struct{})}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:      &config.Config{VPNs: map[string]config.VPNConfig{"work": {Type: "tailscale"}}},
		Adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"work": adapter}),
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.Connect(ctx, "work"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Connect = %v, want cancellation", err)
	}
	select {
	case <-adapter.started:
		t.Fatal("adapter started after cancellation")
	default:
	}
}

type disappearingAdapter struct {
	*blockingAdapter
	reads atomic.Int32
}

func (a *disappearingAdapter) Status() vpn.Status {
	if a.reads.Add(1) == 1 {
		return vpn.StatusConnected
	}
	return vpn.StatusDisconnected
}

func TestConnectAttachedDoesNotStartDisappearingTunnel(t *testing.T) {
	adapter := &disappearingAdapter{blockingAdapter: &blockingAdapter{started: make(chan struct{})}}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:      &config.Config{VPNs: map[string]config.VPNConfig{"work": {Type: "tailscale"}}},
		Adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"work": adapter}),
	})
	if err := svc.ConnectAttached(t.Context(), "work"); !errors.Is(err, ErrTunnelNotAttached) {
		t.Fatalf("ConnectAttached = %v, want ErrTunnelNotAttached", err)
	}
	select {
	case <-adapter.started:
		t.Fatal("attach-only restore started an adapter")
	default:
	}
}

// blockingAdapter's Connect waits like a provider adapter waiting for the
// user in the app: it only returns when its context ends.
type blockingAdapter struct {
	mu       sync.Mutex
	status   vpn.Status
	started  chan struct{}
	returned atomic.Bool // set inside Connect, before it returns
}

func (a *blockingAdapter) Name() string                   { return "blocking" }
func (a *blockingAdapter) Version() string                { return "0" }
func (a *blockingAdapter) Capabilities() vpn.Capabilities { return vpn.Capabilities{} }
func (a *blockingAdapter) Reconnect(context.Context) error {
	return nil
}
func (a *blockingAdapter) TunnelInfo() (*vpn.TunnelInfo, error) { return nil, nil }
func (a *blockingAdapter) set(s vpn.Status) {
	a.mu.Lock()
	a.status = s
	a.mu.Unlock()
}
func (a *blockingAdapter) Status() vpn.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status.Normalize()
}
func (a *blockingAdapter) Connect(ctx context.Context, _ vpn.AdapterConfig) error {
	defer a.returned.Store(true)
	a.set(vpn.StatusConnecting)
	close(a.started)
	<-ctx.Done()
	a.set(vpn.StatusDisconnected)
	return ctx.Err()
}
func (a *blockingAdapter) Disconnect(context.Context) error {
	a.set(vpn.StatusDisconnected)
	return nil
}

type recordingWatchdog struct {
	mu    sync.Mutex
	calls []string
}

func (w *recordingWatchdog) MarkIntended(name string) { w.add("intended:" + name) }
func (w *recordingWatchdog) MarkActive(name string)   { w.add("active:" + name) }
func (w *recordingWatchdog) add(c string) {
	w.mu.Lock()
	w.calls = append(w.calls, c)
	w.mu.Unlock()
}

// `down` while a connect is still running (e.g. waiting for the user in
// FortiClient) cancels it and returns only once it has stopped: the
// status is final at once and the watchdog is never re-armed.
func TestDisconnect_CancelsInflightConnect(t *testing.T) {
	adapter := &blockingAdapter{status: vpn.StatusDisconnected, started: make(chan struct{})}
	wd := &recordingWatchdog{}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:      &config.Config{VPNs: map[string]config.VPNConfig{"forti": {Type: "tailscale"}}},
		Adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"forti": adapter}),
		Watchdog: wd,
	})

	connectErr := make(chan error, 1)
	go func() { connectErr <- svc.Connect(context.Background(), "forti") }()
	select {
	case <-adapter.started:
	case <-time.After(5 * time.Second):
		t.Fatal("connect never reached the adapter")
	}
	if st := adapter.Status(); st != vpn.StatusConnecting {
		t.Fatalf("status while connecting = %s", st)
	}

	if err := svc.Disconnect(context.Background(), "forti"); err != nil {
		t.Fatal(err)
	}
	// The adapter's Connect must already have returned when Disconnect did.
	// Checked inside the adapter: the goroutine above may not have sent on
	// connectErr yet (scheduler timing, seen on Windows CI).
	if !adapter.returned.Load() {
		t.Fatal("Disconnect returned while the connect was still running")
	}
	select {
	case err := <-connectErr:
		if err == nil {
			t.Fatal("cancelled connect reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Connect never returned")
	}
	if st := adapter.Status(); st != vpn.StatusDisconnected {
		t.Fatalf("status right after down = %s, want disconnected", st)
	}
	wd.mu.Lock()
	defer wd.mu.Unlock()
	for _, c := range wd.calls {
		if c == "active:forti" {
			t.Fatalf("watchdog re-armed after down: %v", wd.calls)
		}
	}
	if len(wd.calls) == 0 || wd.calls[len(wd.calls)-1] != "intended:forti" {
		t.Fatalf("watchdog calls = %v, want the intent recorded last", wd.calls)
	}
}
