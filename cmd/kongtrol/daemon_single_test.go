package main

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// stubDaemonFns swaps the daemon-client indirections used by runDown /
// runUpAttached and restores them (plus the globals they read) afterwards.
func stubDaemonFns(t *testing.T, probe bool, active []string) (disconnected *[]string, stopped *int) {
	t.Helper()
	oldProbe, oldDisc, oldActive, oldStop := daemonProbeFn, daemonDisconnectFn, daemonActiveProfilesFn, stopDaemonFn
	oldCfg, oldAdapters, oldAll, oldGroup := cfg, adapters, downAll, downGroup
	t.Cleanup(func() {
		daemonProbeFn, daemonDisconnectFn, daemonActiveProfilesFn, stopDaemonFn = oldProbe, oldDisc, oldActive, oldStop
		cfg, adapters, downAll, downGroup = oldCfg, oldAdapters, oldAll, oldGroup
	})

	cfg = &config.Config{}
	cfg.Monitor.Dashboard.Bind = "127.0.0.1"
	cfg.Monitor.Dashboard.Port = 1
	// runDown validates names against the config before any work.
	cfg.VPNs = map[string]config.VPNConfig{"office": {}, "home": {}}

	var mu sync.Mutex
	disc := []string{}
	n := 0
	daemonProbeFn = func(string) bool { return probe }
	daemonDisconnectFn = func(_ string, name string) error {
		mu.Lock()
		defer mu.Unlock()
		disc = append(disc, name)
		return nil
	}
	daemonActiveProfilesFn = func(string) ([]string, error) { return active, nil }
	stopDaemonFn = func() { n++ }
	return &disc, &n
}

func TestRunDown_ViaDaemon_KeepsDaemonWhileOthersActive(t *testing.T) {
	disc, stopped := stubDaemonFns(t, true, []string{"office", "home"})
	downAll = false
	if err := runDown(context.Background(), []string{"office"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*disc, []string{"office"}) {
		t.Fatalf("expected disconnect through daemon API for office, got %v", *disc)
	}
	if *stopped != 0 {
		t.Fatal("daemon must keep running while another profile (home) is still active")
	}
}

func TestRunDown_ViaDaemon_StopsWhenLastProfileGone(t *testing.T) {
	// The daemon snapshot may still list office (stale by one poll); it is
	// discounted because we just disconnected it.
	disc, stopped := stubDaemonFns(t, true, []string{"office"})
	downAll = false
	if err := runDown(context.Background(), []string{"office"}); err != nil {
		t.Fatal(err)
	}
	if len(*disc) != 1 || *stopped != 1 {
		t.Fatalf("disc=%v stopped=%d, want one daemon disconnect and one stop", *disc, *stopped)
	}
}

func TestRunDown_All_ViaDaemon(t *testing.T) {
	disc, stopped := stubDaemonFns(t, true, []string{"a", "b"})
	adapters = map[string]vpn.VPNAdapter{"a": &mockAdapter{}, "b": &mockAdapter{}}
	downAll = true
	if err := runDown(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	got := append([]string(nil), *disc...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a", "b"}) || *stopped != 1 {
		t.Fatalf("disc=%v stopped=%d, want both via daemon and one stop", got, *stopped)
	}
}

func TestRunDown_NoDaemon_UsesLocalAndStops(t *testing.T) {
	disc, stopped := stubDaemonFns(t, false, nil)
	adapters = map[string]vpn.VPNAdapter{}
	downAll = true
	if err := runDown(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(*disc) != 0 || *stopped != 1 {
		t.Fatalf("disc=%v stopped=%d, want no API calls and one stop", *disc, *stopped)
	}
}

func TestDisconnectActiveOnShutdown_OnlyActiveProfiles(t *testing.T) {
	ads := map[string]vpn.VPNAdapter{
		"up":   &mockAdapter{status: vpn.StatusConnected},
		"half": &mockAdapter{status: vpn.StatusConnecting},
		"down": &mockAdapter{status: vpn.StatusDisconnected},
		"zero": &mockAdapter{}, // "" normalizes to disconnected
	}
	var mu sync.Mutex
	var got []string
	disconnectActiveOnShutdown(ads, func(_ context.Context, name string) error {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, name)
		return nil
	}, time.Second)
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"half", "up"}) {
		t.Fatalf("disconnected %v, want [half up]", got)
	}
}

func TestDisconnectActiveOnShutdown_BoundedByTimeout(t *testing.T) {
	ads := map[string]vpn.VPNAdapter{"stuck": &mockAdapter{status: vpn.StatusConnected}}
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	start := time.Now()
	disconnectActiveOnShutdown(ads, func(context.Context, string) error {
		<-release // ignores ctx, like a wedged adapter
		return nil
	}, 200*time.Millisecond)
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("shutdown disconnect took %v, must be bounded by its timeout", el)
	}
}
