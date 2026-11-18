package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

var t0 = time.Unix(1_800_000_000, 0)

// obs is one counters read: at seconds after t0, handshake hsAgo seconds
// before that read (-1 = never), rx/tx totals, keepalive seconds.
type obs struct {
	at        int
	hsAgo     int
	rx, tx    uint64
	keepalive int
	err       error
}

func stats(o obs) privops.WGStats {
	at := t0.Add(time.Duration(o.at) * time.Second)
	var hs int64
	if o.hsAgo >= 0 {
		hs = at.Add(-time.Duration(o.hsAgo) * time.Second).Unix()
	}
	return privops.WGStats{Peers: []privops.WGPeerStats{{PublicKey: "k", LatestHandshake: hs,
		RxBytes: o.rx, TxBytes: o.tx, PersistentKeepalive: o.keepalive}}}
}

func TestHealthTracker(t *testing.T) {
	cases := []struct {
		name string
		obs  []obs
		want vpn.HealthState
		key  string
	}{
		{"fresh handshake", []obs{{at: 60, hsAgo: 10, rx: 100, tx: 100}}, vpn.HealthOK, ""},
		{"fresh handshake with keepalive", []obs{{at: 60, hsAgo: 100, rx: 1, tx: 1, keepalive: 25}}, vpn.HealthOK, ""},
		{"idle, no keepalive, no traffic", []obs{
			{at: 0, hsAgo: 600, rx: 500, tx: 500},
			{at: 60, hsAgo: 660, rx: 500, tx: 500},
			{at: 120, hsAgo: 720, rx: 500, tx: 500},
		}, vpn.HealthIdle, ""},
		{"never handshaked, unused, no keepalive", []obs{
			{at: 0, hsAgo: -1}, {at: 60, hsAgo: -1},
		}, vpn.HealthIdle, ""},
		{"keepalive, stale handshake", []obs{
			{at: 0, hsAgo: 300, rx: 10, tx: 10, keepalive: 25},
			{at: 40, hsAgo: 340, rx: 10, tx: 400, keepalive: 25},
		}, vpn.HealthNoResponse, ""},
		{"keepalive, stale, inside grace", []obs{
			{at: 0, hsAgo: 300, rx: 10, tx: 10, keepalive: 25},
			{at: 20, hsAgo: 320, rx: 10, tx: 200, keepalive: 25},
		}, vpn.HealthIdle, ""},
		{"tx without rx, stale", []obs{
			{at: 0, hsAgo: 900, rx: 1000, tx: 1000},
			{at: 30, hsAgo: 930, rx: 1000, tx: 1300},
			{at: 40, hsAgo: 940, rx: 1000, tx: 1600},
			{at: 50, hsAgo: 950, rx: 1000, tx: 1900},
		}, vpn.HealthNoResponse, ""},
		{"tx without rx, too short to judge", []obs{
			{at: 30, hsAgo: 900, rx: 1000, tx: 1000},
			{at: 40, hsAgo: 910, rx: 1000, tx: 1300},
			{at: 50, hsAgo: 920, rx: 1000, tx: 1600},
		}, vpn.HealthIdle, ""},
		{"traffic resumes and server answers", []obs{
			{at: 0, hsAgo: 900, rx: 1000, tx: 1000},
			{at: 30, hsAgo: 930, rx: 1000, tx: 1300},
			{at: 40, hsAgo: 1, rx: 1400, tx: 1600},
		}, vpn.HealthOK, ""},
		{"rx grows while handshake stays stale", []obs{
			{at: 0, hsAgo: 900, rx: 1000, tx: 1000},
			{at: 30, hsAgo: 930, rx: 1000, tx: 1300},
			{at: 40, hsAgo: 940, rx: 1100, tx: 1600},
			{at: 50, hsAgo: 950, rx: 1100, tx: 1600},
		}, vpn.HealthIdle, ""},
		{"never handshaked, sent initiations", []obs{
			{at: 0, hsAgo: -1, tx: 148}, {at: 35, hsAgo: -1, tx: 1036},
		}, vpn.HealthNoResponse, ""},
		{"never handshaked, keepalive", []obs{
			{at: 0, hsAgo: -1, keepalive: 25}, {at: 31, hsAgo: -1, keepalive: 25},
		}, vpn.HealthNoResponse, ""},
		{"never handshaked, inside grace", []obs{
			{at: 0, hsAgo: -1, tx: 148}, {at: 10, hsAgo: -1, tx: 296},
		}, vpn.HealthIdle, ""},
		{"helper outdated", []obs{{at: 0, err: privops.ErrHelperOutdated}}, vpn.HealthUnknown, "service.error.outdated"},
		{"helper missing", []obs{{at: 0, err: privops.ErrHelperNotInstalled}}, vpn.HealthUnknown, "service.error.not_installed"},
		{"other read error", []obs{{at: 0, err: errors.New("boom")}}, vpn.HealthUnknown, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tr healthTracker
			tr.reset(t0)
			var h vpn.TunnelHealth
			for _, o := range tc.obs {
				h = tr.observe(t0.Add(time.Duration(o.at)*time.Second), stats(o), o.err)
			}
			if h.State != tc.want || h.ReasonKey != tc.key {
				t.Fatalf("health = %+v, want %s (%q)", h, tc.want, tc.key)
			}
		})
	}
}

// Since stays while the state holds; the newest peer handshake counts.
func TestHealthTracker_SinceAndPeers(t *testing.T) {
	var tr healthTracker
	tr.reset(t0)
	st := privops.WGStats{Peers: []privops.WGPeerStats{
		{PublicKey: "a", LatestHandshake: t0.Add(-time.Hour).Unix()},
		{PublicKey: "b", LatestHandshake: t0.Add(50 * time.Second).Unix(), RxBytes: 5, TxBytes: 7},
	}}
	h1 := tr.observe(t0.Add(60*time.Second), st, nil)
	h2 := tr.observe(t0.Add(70*time.Second), st, nil)
	if h1.State != vpn.HealthOK || !h2.Since.Equal(h1.Since) || !h2.LastHandshake.Equal(t0.Add(50*time.Second)) {
		t.Fatalf("h1 %+v h2 %+v", h1, h2)
	}
	if rx, tx, ok := tr.totals(); !ok || rx != 5 || tx != 7 {
		t.Fatalf("totals %d %d %v", rx, tx, ok)
	}
}

// existingIface returns the name of an interface present on this machine
// (Status checks that the tunnel interface exists).
func existingIface(t *testing.T) string {
	t.Helper()
	ifs, err := net.Interfaces()
	if err != nil || len(ifs) == 0 {
		t.Skip("no network interfaces")
	}
	return ifs[0].Name
}

// Issue #33: an interface that exists but whose server stopped answering
// is no longer reported as connected; it recovers when handshakes resume,
// and a stats read failure is "unknown", not a silent "connected".
func TestStatus_FollowsHandshake(t *testing.T) {
	iface := existingIface(t)
	clock := t0
	oldNow, oldStats := now, wgStatsFunc
	t.Cleanup(func() { now, wgStatsFunc = oldNow, oldStats })
	now = func() time.Time { return clock }
	var next obs
	wgStatsFunc = func(string, string) (privops.WGStats, error) {
		if next.err != nil {
			return privops.WGStats{}, next.err
		}
		return stats(next), nil
	}
	a := &Adapter{ifaceName: iface}

	step := func(o obs) vpn.Status {
		clock = t0.Add(time.Duration(o.at) * time.Second)
		next = o
		return a.Status()
	}
	if s := step(obs{at: 0, hsAgo: 5, rx: 10, tx: 10, keepalive: 25}); s != vpn.StatusConnected {
		t.Fatalf("fresh: %s", s)
	}
	step(obs{at: 300, hsAgo: 250, rx: 10, tx: 500, keepalive: 25})
	if s := a.Status(); s != vpn.StatusError {
		t.Fatalf("stale with keepalive: %s", s)
	}
	if h := a.TunnelHealth(); h == nil || h.State != vpn.HealthNoResponse {
		t.Fatalf("health = %+v", h)
	}
	// Within statsEvery the cached verdict holds.
	if s := step(obs{at: 305, hsAgo: 1, rx: 99, tx: 600, keepalive: 25}); s != vpn.StatusError {
		t.Fatalf("cached: %s", s)
	}
	if s := step(obs{at: 320, hsAgo: 1, rx: 99, tx: 600, keepalive: 25}); s != vpn.StatusConnected {
		t.Fatalf("recovered: %s", s)
	}
	if s := step(obs{at: 340, err: fmt.Errorf("x: %w", privops.ErrHelperOutdated)}); s != vpn.StatusConnected {
		t.Fatalf("unknown: %s", s)
	}
	if h := a.TunnelHealth(); h == nil || h.State != vpn.HealthUnknown || h.ReasonKey != "service.error.outdated" {
		t.Fatalf("unknown health = %+v", h)
	}
}

// A reconnect of a tunnel that stopped answering replaces it instead of
// reusing the running one.
func TestConnect_RestartsUnresponsiveTunnel(t *testing.T) {
	a := &Adapter{status: vpn.StatusError, health: &vpn.TunnelHealth{State: vpn.HealthNoResponse}}
	rec := &installOps{}
	old := opsFunc
	defer func() { opsFunc = old }()
	opsFunc = func(context.Context) (privops.Ops, error) { return rec, nil }
	// The config path does not exist: Connect fails after wgUp, which is
	// all this test needs to observe.
	_ = a.Connect(context.Background(), vpn.AdapterConfig{ConfigPath: "Z:/nope/kongtrol-test-restart.conf"})
	if rec.installs != 1 {
		t.Fatalf("installs = %d, want 1 (restart)", rec.installs)
	}
}

type installOps struct {
	privops.Ops
	installs int
}

func (r *installOps) WGInstallTunnel(context.Context, privops.WGInstallRequest) (privops.WGTunnel, error) {
	r.installs++
	return privops.WGTunnel{}, errors.New("stop here")
}
