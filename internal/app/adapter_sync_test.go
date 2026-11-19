package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type syncTestAdapter struct {
	typ    string
	status atomic.Value
}

func newSyncTestAdapter(typ string) *syncTestAdapter {
	a := &syncTestAdapter{typ: typ}
	a.status.Store(vpn.StatusDisconnected)
	return a
}

func (a *syncTestAdapter) set(s vpn.Status)                                 { a.status.Store(s) }
func (a *syncTestAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (a *syncTestAdapter) Disconnect(context.Context) error                 { return nil }
func (a *syncTestAdapter) Reconnect(context.Context) error                  { return nil }
func (a *syncTestAdapter) Status() vpn.Status                               { return a.status.Load().(vpn.Status) }
func (a *syncTestAdapter) TunnelInfo() (*vpn.TunnelInfo, error)             { return nil, nil }
func (a *syncTestAdapter) Name() string                                     { return a.typ }
func (a *syncTestAdapter) Version() string                                  { return "0" }
func (a *syncTestAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }

func syncCfg(vpns map[string]config.VPNConfig) *config.Config {
	return &config.Config{VPNs: vpns}
}

func newTestSync(t *testing.T, cfg *config.Config) (*AdapterSync, *vpn.Set, *[]string) {
	t.Helper()
	initial := map[string]vpn.VPNAdapter{}
	for name, vc := range cfg.VPNs {
		initial[name] = newSyncTestAdapter(vc.Type)
	}
	set := vpn.NewSet(initial)
	var forgotten []string
	s := NewAdapterSync(set, cfg, func(n string) { forgotten = append(forgotten, n) })
	s.SetBuilder(func(vc config.VPNConfig) (vpn.VPNAdapter, error) {
		if vc.Type == "bogus" {
			return nil, errors.New("unknown type")
		}
		return newSyncTestAdapter(vc.Type), nil
	})
	return s, set, &forgotten
}

func TestAdapterSync_AddRemoveReplace(t *testing.T) {
	cfg := syncCfg(map[string]config.VPNConfig{
		"office": {Type: "openvpn", ConfigFile: "a.ovpn"},
		"home":   {Type: "wireguard", ConfigFile: "h.conf"},
	})
	s, set, forgotten := newTestSync(t, cfg)
	before, _ := set.Get("home")

	next := syncCfg(map[string]config.VPNConfig{
		"office": {Type: "openvpn", ConfigFile: "b.ovpn"}, // edited
		"home":   {Type: "wireguard", ConfigFile: "h.conf", Priority: 5},
		"new":    {Type: "surfshark"},
	})
	res := s.Sync(next)
	if !slices.Equal(res.Added, []string{"new"}) || !slices.Equal(res.Replaced, []string{"office"}) || len(res.Removed) != 0 {
		t.Fatalf("res = %+v", res)
	}
	if after, _ := set.Get("home"); after != before {
		t.Fatal("a change that is not part of the adapter seed must keep the adapter")
	}
	if !set.Has("new") || !res.Changed() {
		t.Fatal("new profile not registered")
	}

	res = s.Sync(syncCfg(map[string]config.VPNConfig{"office": next.VPNs["office"]}))
	if !slices.Equal(res.Removed, []string{"home", "new"}) || set.Has("home") {
		t.Fatalf("remove: %+v", res)
	}
	if !slices.Equal(*forgotten, []string{"home", "new"}) {
		t.Fatalf("watchdog must forget removed profiles: %v", *forgotten)
	}
}

func TestAdapterSync_BusyProfilesAreNotTouched(t *testing.T) {
	cfg := syncCfg(map[string]config.VPNConfig{
		"office": {Type: "openvpn", ConfigFile: "a.ovpn"},
		"home":   {Type: "wireguard", ConfigFile: "h.conf"},
	})
	s, set, _ := newTestSync(t, cfg)
	office, _ := set.Get("office")
	office.(*syncTestAdapter).set(vpn.StatusConnected)
	home, _ := set.Get("home")
	home.(*syncTestAdapter).set(vpn.StatusConnecting)

	edited := syncCfg(map[string]config.VPNConfig{"office": {Type: "openvpn", ConfigFile: "b.ovpn"}})
	res := s.Sync(edited)
	if !slices.Equal(res.Reconnect, []string{"office"}) || !slices.Equal(res.Kept, []string{"home"}) {
		t.Fatalf("res = %+v", res)
	}
	if cur, _ := set.Get("office"); cur != office {
		t.Fatal("an up tunnel's adapter must stay in place")
	}
	if !set.Has("home") {
		t.Fatal("a connecting profile must not be removed")
	}

	// Once down, Resync applies the pending edit and the removal.
	office.(*syncTestAdapter).set(vpn.StatusDisconnected)
	home.(*syncTestAdapter).set(vpn.StatusDisconnected)
	res = s.Resync()
	if !slices.Equal(res.Replaced, []string{"office"}) || !slices.Equal(res.Removed, []string{"home"}) {
		t.Fatalf("resync = %+v", res)
	}
	if cur, _ := set.Get("office"); cur == office {
		t.Fatal("edit not applied after disconnect")
	}

	// The busy hook covers a connect that has not changed the status yet.
	s.SetBusy(func(n string) bool { return n == "office" })
	res = s.Sync(syncCfg(map[string]config.VPNConfig{"office": {Type: "openvpn", ConfigFile: "c.ovpn"}}))
	if !slices.Equal(res.Reconnect, []string{"office"}) {
		t.Fatalf("busy hook ignored: %+v", res)
	}
}

func TestAdapterSync_FailedBuild(t *testing.T) {
	s, set, _ := newTestSync(t, syncCfg(map[string]config.VPNConfig{"office": {Type: "openvpn"}}))
	res := s.Sync(syncCfg(map[string]config.VPNConfig{"office": {Type: "openvpn"}, "x": {Type: "bogus"}}))
	if res.Failed["x"] == nil || set.Has("x") {
		t.Fatalf("res = %+v", res)
	}
}

// TestAdapterSync_ConcurrentWithReaders: syncs while readers walk the set
// (collector / watchdog / API); meaningful under -race.
func TestAdapterSync_ConcurrentWithReaders(t *testing.T) {
	s, set, _ := newTestSync(t, syncCfg(map[string]config.VPNConfig{"office": {Type: "openvpn"}}))
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, a := range set.Snapshot() {
					_ = a.Status()
				}
				_, _ = set.Get("p3")
			}
		})
	}
	for i := range 100 {
		vpns := map[string]config.VPNConfig{"office": {Type: "openvpn"}}
		for j := range i % 7 {
			vpns[fmt.Sprintf("p%d", j)] = config.VPNConfig{Type: "wireguard", ConfigFile: fmt.Sprintf("%d.conf", i)}
		}
		if i%2 == 0 {
			s.Sync(syncCfg(vpns))
		} else {
			s.Resync()
		}
	}
	close(stop)
	wg.Wait()
}
