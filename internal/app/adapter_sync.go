package app

import (
	"reflect"
	"sort"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// NewProfileAdapter builds the adapter of one profile and pre-seeds its
// config, so Status() can find a tunnel started outside Kongtrol (attach
// mode) before the first Connect. Used at boot and by AdapterSync.
func NewProfileAdapter(vc config.VPNConfig) (vpn.VPNAdapter, error) {
	a, err := vpn.New(vc.Type)
	if err != nil {
		return nil, err
	}
	if c, ok := a.(vpn.Configurable); ok {
		c.Configure(adapterSeed(vc))
	}
	return a, nil
}

// adapterSeed is the part of a profile an adapter is built from. Two
// profiles with the same type and seed need no new adapter; every other
// field (failover, priority, kill switch, policies) is read from the
// config when it is used.
func adapterSeed(vc config.VPNConfig) vpn.AdapterConfig {
	return vpn.AdapterConfig{
		Host:       vc.Host,
		Port:       vc.Port,
		TunnelName: vc.TunnelName,
		CertPath:   vc.Auth.Cert,
		KeyPath:    vc.Auth.Key,
		ConfigPath: vc.ConfigFile,
		Username:   vc.Auth.Username,
		Extra:      vc.AdapterExtra(),
	}
}

type adapterKey struct {
	typ  string
	seed vpn.AdapterConfig
}

func keyOf(vc config.VPNConfig) adapterKey {
	return adapterKey{typ: vc.Type, seed: adapterSeed(vc)}
}

// SyncResult says what AdapterSync.Sync did.
type SyncResult struct {
	Added    []string // new profiles, ready to connect
	Removed  []string // profiles gone from the config, adapter dropped
	Replaced []string // edited while down: new adapter in place
	// Reconnect lists profiles edited while their tunnel is up (or coming
	// up): the running tunnel keeps the old settings until the user
	// reconnects it; the new adapter is put in place once it is down.
	Reconnect []string
	// Kept lists profiles removed from the config while their tunnel is
	// up (a hand edit; the API refuses it): they stay until disconnected.
	Kept []string
	// Failed maps a profile to the error building its adapter (an
	// unknown type in a hand edit); a daemon restart reports it the same.
	Failed map[string]error
}

// Changed reports whether the set changed.
func (r SyncResult) Changed() bool {
	return len(r.Added)+len(r.Removed)+len(r.Replaced) > 0
}

// Has reports whether name is in list.
func Has(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

// AdapterSync keeps the daemon's adapter set in line with the config, so
// profiles created, edited, renamed or deleted take effect without a
// restart (ADR-0053). A tunnel that is up is never touched: its adapter
// owns the running process or session.
type AdapterSync struct {
	mu      sync.Mutex
	set     *vpn.Set
	build   func(config.VPNConfig) (vpn.VPNAdapter, error)
	busy    func(name string) bool
	forget  func(name string)
	applied map[string]adapterKey
	last    *config.Config
}

// NewAdapterSync returns a sync for set, whose adapters were built from
// cfg. forget (optional) is called for every profile removed,
// so the watchdog drops what it knew about it.
func NewAdapterSync(set *vpn.Set, cfg *config.Config, forget func(string)) *AdapterSync {
	s := &AdapterSync{
		set:     set,
		build:   NewProfileAdapter,
		forget:  forget,
		applied: map[string]adapterKey{},
		last:    cfg,
	}
	if cfg != nil {
		for name, vc := range cfg.VPNs {
			if set.Has(name) {
				s.applied[name] = keyOf(vc)
			}
		}
	}
	return s
}

// SetBuilder replaces the adapter constructor (tests).
func (s *AdapterSync) SetBuilder(fn func(config.VPNConfig) (vpn.VPNAdapter, error)) {
	s.mu.Lock()
	s.build = fn
	s.mu.Unlock()
}

// SetBusy adds a check for profiles that are busy although their adapter
// reports disconnected (a connect just starting).
func (s *AdapterSync) SetBusy(fn func(string) bool) {
	s.mu.Lock()
	s.busy = fn
	s.mu.Unlock()
}

func (s *AdapterSync) isBusy(name string, a vpn.VPNAdapter) bool {
	if a != nil && a.Status().Normalize() != vpn.StatusDisconnected {
		return true
	}
	return s.busy != nil && s.busy(name)
}

// Sync applies cfg to the set and remembers it for Resync.
func (s *AdapterSync) Sync(cfg *config.Config) SyncResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = cfg
	return s.syncLocked(cfg)
}

// Resync applies the last config again: called after a disconnect, so an
// edit made while the tunnel was up takes effect now.
func (s *AdapterSync) Resync() SyncResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.syncLocked(s.last)
}

func (s *AdapterSync) syncLocked(cfg *config.Config) SyncResult {
	var res SyncResult
	if cfg == nil {
		return res
	}
	var forgotten []string
	for _, name := range s.set.Names() {
		if _, ok := cfg.VPNs[name]; ok {
			continue
		}
		a, _ := s.set.Get(name)
		if s.isBusy(name, a) {
			res.Kept = append(res.Kept, name)
			continue
		}
		s.set.Delete(name)
		delete(s.applied, name)
		forgotten = append(forgotten, name)
		res.Removed = append(res.Removed, name)
	}
	names := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		vc := cfg.VPNs[name]
		key := keyOf(vc)
		cur, exists := s.set.Get(name)
		if exists && reflect.DeepEqual(s.applied[name], key) {
			continue
		}
		if exists && s.isBusy(name, cur) {
			res.Reconnect = append(res.Reconnect, name)
			continue
		}
		a, err := s.build(vc)
		if err != nil {
			if res.Failed == nil {
				res.Failed = map[string]error{}
			}
			res.Failed[name] = err
			continue
		}
		s.set.Put(name, a)
		s.applied[name] = key
		if exists {
			// The watchdog looks the adapter up on every poll, so a
			// reconnect loop for this profile follows the new one.
			res.Replaced = append(res.Replaced, name)
		} else {
			res.Added = append(res.Added, name)
		}
	}
	if s.forget != nil {
		for _, name := range forgotten {
			s.forget(name)
		}
	}
	return res
}
