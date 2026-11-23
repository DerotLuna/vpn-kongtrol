package app

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Regression tests for #46 (ADR-0097): a disconnect the user asked for is
// never a "drop" — no kill switch block, no "se cayó" notice, no watchdog
// reconnect — even when the kill switch looks at the tunnel while the
// adapter is still going down.

// slowDownAdapter stays connected during Disconnect until duringDown has
// run (the kill switch's 5 s tick landing mid-disconnect, like OpenVPN
// waiting for the helper to stop openvpn), then reports disconnected.
type slowDownAdapter struct {
	mu         sync.Mutex
	status     vpn.Status
	duringDown func()
	connects   atomic.Int32
}

func (a *slowDownAdapter) Name() string                   { return "slow" }
func (a *slowDownAdapter) Version() string                { return "" }
func (a *slowDownAdapter) Capabilities() vpn.Capabilities { return vpn.Capabilities{} }
func (a *slowDownAdapter) Reconnect(context.Context) error {
	return nil
}
func (a *slowDownAdapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	return &vpn.TunnelInfo{InterfaceName: "tun0"}, nil
}
func (a *slowDownAdapter) set(s vpn.Status) { a.mu.Lock(); a.status = s; a.mu.Unlock() }
func (a *slowDownAdapter) Status() vpn.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status.Normalize()
}
func (a *slowDownAdapter) Connect(context.Context, vpn.AdapterConfig) error {
	a.connects.Add(1)
	a.set(vpn.StatusConnected)
	return nil
}
func (a *slowDownAdapter) Disconnect(context.Context) error {
	if a.duringDown != nil {
		a.duringDown()
	}
	a.set(vpn.StatusDisconnected)
	return nil
}

// staleReadAdapter runs onRead inside Status before answering: the kill
// switch's status read happens before the user's intent is recorded, and
// the rest of that Apply runs after it.
type staleReadAdapter struct {
	fakeAdapter
	onRead func()
}

func (a *staleReadAdapter) Status() vpn.Status {
	if fn := a.onRead; fn != nil {
		a.onRead = nil
		fn()
	}
	return a.fakeAdapter.Status()
}

func TestKillSwitch_ApplyDuringUserDisconnectIsNotADrop(t *testing.T) {
	ad := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}
	s, b := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"brainbox": ad},
		[]routing.Route{rt("10.20.0.0/16", "tun0")},
		fakeDests{"profile:brainbox": pfx("160.79.104.10/32")})
	notices := 0
	s.SetOnBlock(func(string, bool) { notices++ })
	_ = s.Apply()

	s.NoteIntent("brainbox", true) // the user pressed Desconectar
	_ = s.Apply()                  // tick while the adapter is still going down
	ad.status = vpn.StatusDisconnected
	_ = s.Apply() // afterDisconnect

	if notices != 0 || len(b.set) != 0 || s.Status().State != "armed" {
		t.Fatalf("user disconnect treated as a drop: notices=%d blocked=%v status=%+v", notices, b.set, s.Status())
	}

	// Brought up again outside Kongtrol (e.g. in the provider app) after
	// it was seen down: protected again, and a later drop is a real one.
	ad.status = vpn.StatusConnected
	_ = s.Apply()
	ad.status = vpn.StatusDisconnected
	_ = s.Apply()
	if notices != 1 || len(b.set) == 0 {
		t.Fatalf("external reconnect then drop: notices=%d blocked=%v", notices, b.set)
	}
}

func TestKillSwitch_StatusReadBeforeIntentIsNotADrop(t *testing.T) {
	ad := &staleReadAdapter{fakeAdapter: fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}}
	s, b := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"brainbox": ad},
		[]routing.Route{rt("10.20.0.0/16", "tun0")}, fakeDests{})
	notices := 0
	s.SetOnBlock(func(string, bool) { notices++ })
	_ = s.Apply()

	ad.onRead = func() { s.NoteIntent("brainbox", true) }
	_ = s.Apply() // read "connected", then the intent lands
	ad.status = vpn.StatusDisconnected
	_ = s.Apply()

	if notices != 0 || len(b.set) != 0 {
		t.Fatalf("stale status read turned a user disconnect into a drop: notices=%d blocked=%v", notices, b.set)
	}
}

// The full path: ProfileService.Disconnect wired like the daemon
// (OnIntent, ApplyKill, BeginDisconnect) with the real watchdog.
func TestDisconnect_UserDisconnectNeverDropBlockOrReconnect(t *testing.T) {
	cfg := ksCfg()
	// No host and a type without preflight: Connect touches no network
	// and no helper.
	cfg.VPNs = map[string]config.VPNConfig{"brainbox": {Type: "tailscale"}}
	ad := &slowDownAdapter{status: vpn.StatusDisconnected}
	set := vpn.NewSet(map[string]vpn.VPNAdapter{"brainbox": ad})
	localSubnets = func() []netip.Prefix { return nil }
	ks := NewKillSwitchService(cfg, set, nil)
	ks.lookupIP = nil
	b := &fakeBlocker{}
	ks.SetBlocker(b)
	ks.SetDestinations(fakeDests{"profile:brainbox": pfx("160.79.104.10/32")})
	ks.SetRouteLister(func() ([]routing.Route, error) { return []routing.Route{rt("10.20.0.0/16", "tun0")}, nil })
	notices := 0
	ks.SetOnBlock(func(string, bool) { notices++ })
	ad.duringDown = func() { _ = ks.Apply() }

	wd := monitor.NewWatchdog(set, func(ctx context.Context, name string) error {
		return ad.Connect(ctx, vpn.AdapterConfig{})
	}, zap.NewNop())
	wd.SetPollInterval(10 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.Start(ctx)
	defer wd.Stop()

	svc := NewProfileService(ProfileServiceDeps{
		Cfg:             cfg,
		Adapters:        set,
		Watchdog:        wd,
		ApplyKill:       ks.Apply,
		BeginDisconnect: ks.BeginDisconnect,
		OnIntent:        ks.NoteIntent,
	})
	if err := svc.Connect(context.Background(), "brainbox"); err != nil {
		t.Fatal(err)
	}
	_ = ks.Apply()
	if err := svc.Disconnect(context.Background(), "brainbox"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // several watchdog polls
	_ = ks.Apply()

	if notices != 0 || len(b.set) != 0 || ks.Status().State != "armed" {
		t.Fatalf("user disconnect treated as a drop: notices=%d blocked=%v status=%+v", notices, b.set, ks.Status())
	}
	if n := ad.connects.Load(); n != 1 {
		t.Fatalf("watchdog reconnected a tunnel the user turned off: %d connects", n)
	}
	if st := ad.Status(); st != vpn.StatusDisconnected {
		t.Fatalf("status after down = %s", st)
	}
}
