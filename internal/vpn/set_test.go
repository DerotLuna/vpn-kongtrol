package vpn

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

type setTestAdapter struct{ id int }

func (a *setTestAdapter) Connect(context.Context, AdapterConfig) error { return nil }
func (a *setTestAdapter) Disconnect(context.Context) error             { return nil }
func (a *setTestAdapter) Reconnect(context.Context) error              { return nil }
func (a *setTestAdapter) Status() Status                               { return StatusDisconnected }
func (a *setTestAdapter) TunnelInfo() (*TunnelInfo, error)             { return nil, nil }
func (a *setTestAdapter) Name() string                                 { return "t" }
func (a *setTestAdapter) Version() string                              { return "0" }
func (a *setTestAdapter) Capabilities() Capabilities                   { return Capabilities{} }

func TestSet_Basics(t *testing.T) {
	src := map[string]VPNAdapter{"b": &setTestAdapter{1}, "a": &setTestAdapter{2}}
	s := NewSet(src)
	src["c"] = &setTestAdapter{3} // the set holds its own copy
	if s.Len() != 2 || s.Has("c") {
		t.Fatalf("NewSet must copy: len=%d", s.Len())
	}
	if got := s.Names(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("Names = %v", got)
	}
	snap := s.Snapshot()
	snap["z"] = nil // changing a snapshot never changes the set
	if s.Has("z") {
		t.Fatal("snapshot aliases the set")
	}
	s.Put("c", &setTestAdapter{3})
	if a, ok := s.Get("c"); !ok || a.(*setTestAdapter).id != 3 {
		t.Fatal("Put/Get")
	}
	if len(snap) != 3 { // a, b, z: taken before Put
		t.Fatalf("old snapshot changed: %v", snap)
	}
	if _, ok := s.Delete("a"); !ok || s.Has("a") {
		t.Fatal("Delete")
	}
	if _, ok := s.Delete("a"); ok {
		t.Fatal("Delete twice")
	}

	var nilSet *Set
	if nilSet.Len() != 0 || nilSet.Has("a") || len(nilSet.Names()) != 0 || len(nilSet.Snapshot()) != 0 {
		t.Fatal("nil set must read as empty")
	}
}

// TestSet_ConcurrentReadersAndWriters runs readers and writers together;
// with -race it proves the set needs no outside lock.
func TestSet_ConcurrentReadersAndWriters(t *testing.T) {
	s := NewSet(nil)
	var wg sync.WaitGroup
	for w := range 4 {
		wg.Go(func() {
			for i := range 200 {
				name := fmt.Sprintf("p%d-%d", w, i%10)
				s.Put(name, &setTestAdapter{i})
				if i%3 == 0 {
					s.Delete(name)
				}
			}
		})
	}
	for range 4 {
		wg.Go(func() {
			for range 500 {
				for name, a := range s.Snapshot() {
					if a == nil || name == "" {
						t.Error("bad entry")
					}
				}
				_ = s.Names()
				_, _ = s.Get("p0-1")
			}
		})
	}
	wg.Wait()
}
