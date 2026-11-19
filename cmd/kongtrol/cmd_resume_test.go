package main

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// stubResume replaces the resume seams for one test.
func stubResume(t *testing.T, saved []string, daemonUp bool) (connected *[]string, spawned *[]string) {
	t.Helper()
	oldTake, oldLoad, oldEnsure := takeResumeFn, loadConfigFn, ensureResumed
	oldProbe, oldConnect, oldCfg, oldQuiet := daemonProbeFn, daemonConnectFn, cfg, outputQuiet
	t.Cleanup(func() {
		takeResumeFn, loadConfigFn, ensureResumed = oldTake, oldLoad, oldEnsure
		daemonProbeFn, daemonConnectFn, cfg, outputQuiet = oldProbe, oldConnect, oldCfg, oldQuiet
	})
	outputQuiet = true
	cfg = &config.Config{VPNs: map[string]config.VPNConfig{"office": {}, "home": {}}}
	takeResumeFn = func(string, time.Time) ([]string, error) { return saved, nil }
	loadConfigFn = func() error { return nil }
	daemonProbeFn = func(string) bool { return daemonUp }
	var c, s []string
	daemonConnectFn = func(_, name string) (bool, error) {
		c = append(c, name)
		return false, nil
	}
	ensureResumed = func(_ context.Context, _ string, targets []string) error {
		s = append(s, targets...)
		return nil
	}
	return &c, &s
}

func TestResume_NothingSavedDoesNothing(t *testing.T) {
	connected, spawned := stubResume(t, nil, false)
	loaded := false
	loadConfigFn = func() error { loaded = true; return errors.New("no config") }
	if err := runResume(context.Background()); err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if loaded || len(*connected) != 0 || len(*spawned) != 0 {
		t.Fatalf("loaded=%v connected=%v spawned=%v, want nothing at all", loaded, *connected, *spawned)
	}
}

func TestResume_NoDaemonStartsReplacementWithKnownProfiles(t *testing.T) {
	connected, spawned := stubResume(t, []string{"gone", "home", "office"}, false)
	if err := runResume(context.Background()); err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if want := []string{"home", "office"}; !slices.Equal(*spawned, want) {
		t.Fatalf("spawned with %v, want %v", *spawned, want)
	}
	if len(*connected) != 0 {
		t.Fatalf("connected via API %v, want none (the new daemon connects)", *connected)
	}
}

func TestResume_RunningDaemonConnectsThroughAPI(t *testing.T) {
	connected, spawned := stubResume(t, []string{"office"}, true)
	if err := runResume(context.Background()); err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if !slices.Equal(*connected, []string{"office"}) || len(*spawned) != 0 {
		t.Fatalf("connected=%v spawned=%v, want [office] via API only", *connected, *spawned)
	}
}

func TestResume_OnlyDeletedProfilesDoesNothing(t *testing.T) {
	connected, spawned := stubResume(t, []string{"gone"}, false)
	if err := runResume(context.Background()); err != nil {
		t.Fatalf("runResume: %v", err)
	}
	if len(*connected) != 0 || len(*spawned) != 0 {
		t.Fatalf("connected=%v spawned=%v, want nothing", *connected, *spawned)
	}
}

func TestRememberConnected(t *testing.T) {
	oldStatuses, oldWrite := daemonTunnelStatusesFn, writeResumeFn
	t.Cleanup(func() { daemonTunnelStatusesFn, writeResumeFn = oldStatuses, oldWrite })
	daemonTunnelStatusesFn = func(string) (map[string]vpn.Status, error) {
		return map[string]vpn.Status{
			"office": vpn.StatusConnected,
			"home":   vpn.StatusConnecting,
			"lab":    vpn.StatusDisconnected,
			"alt":    vpn.StatusConnected,
		}, nil
	}
	var got []string
	calls := 0
	writeResumeFn = func(_ string, profiles []string, _ time.Time) error {
		calls++
		got = profiles
		return nil
	}

	rememberConnected("http://x", true)
	if want := []string{"alt", "office"}; !slices.Equal(got, want) {
		t.Fatalf("remembered %v, want %v (connected only)", got, want)
	}

	// No daemon: the list is cleared, never left stale.
	rememberConnected("http://x", false)
	if calls != 2 || len(got) != 0 {
		t.Fatalf("calls=%d remembered=%v, want an empty write", calls, got)
	}
}
