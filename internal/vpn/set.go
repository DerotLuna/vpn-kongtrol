package vpn

import (
	"sort"
	"sync"
	"sync/atomic"
)

// Set is the daemon's live set of profile adapters, keyed by profile name.
// It is safe for concurrent use: readers (collector, watchdog, kill switch,
// API handlers, ProfileService) never lock; writers (profile create, edit,
// rename, delete applied without a restart, ADR-0053) copy the map and swap
// it atomically (copy-on-write). Profiles are few and change rarely, so a
// copy per write is cheaper than a lock on every read.
//
// A nil *Set behaves as an empty, read-only set.
type Set struct {
	mu sync.Mutex // serializes writers; readers only load the pointer
	m  atomic.Pointer[map[string]VPNAdapter]
}

// NewSet returns a set holding a copy of m (nil m gives an empty set).
func NewSet(m map[string]VPNAdapter) *Set {
	s := &Set{}
	cp := make(map[string]VPNAdapter, len(m))
	for k, v := range m {
		cp[k] = v
	}
	s.m.Store(&cp)
	return s
}

func (s *Set) load() map[string]VPNAdapter {
	if s == nil {
		return nil
	}
	if p := s.m.Load(); p != nil {
		return *p
	}
	return nil
}

// Get returns the adapter of profile name.
func (s *Set) Get(name string) (VPNAdapter, bool) {
	a, ok := s.load()[name]
	return a, ok
}

// Has reports whether profile name has an adapter.
func (s *Set) Has(name string) bool {
	_, ok := s.load()[name]
	return ok
}

// Len returns the number of profiles.
func (s *Set) Len() int { return len(s.load()) }

// Snapshot returns a copy of the current name → adapter map. Changes to the
// set after the call are not reflected in it, and changing it does not
// change the set.
func (s *Set) Snapshot() map[string]VPNAdapter {
	cur := s.load()
	out := make(map[string]VPNAdapter, len(cur))
	for k, v := range cur {
		out[k] = v
	}
	return out
}

// Names returns the profile names, sorted.
func (s *Set) Names() []string {
	cur := s.load()
	out := make([]string, 0, len(cur))
	for k := range cur {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Put adds or replaces the adapter of profile name.
func (s *Set) Put(name string, a VPNAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.load()
	next := make(map[string]VPNAdapter, len(cur)+1)
	for k, v := range cur {
		next[k] = v
	}
	next[name] = a
	s.m.Store(&next)
}

// Delete removes profile name and returns the adapter it had.
func (s *Set) Delete(name string) (VPNAdapter, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.load()
	a, ok := cur[name]
	if !ok {
		return nil, false
	}
	next := make(map[string]VPNAdapter, len(cur))
	for k, v := range cur {
		if k != name {
			next[k] = v
		}
	}
	s.m.Store(&next)
	return a, true
}
