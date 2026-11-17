package privops

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a manually advanced clock.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *fakeClock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newClock() *fakeClock { return &fakeClock{t: time.Unix(1_700_000_000, 0)} }

func TestLeaseTable_ExpiresOnlyUnrenewed(t *testing.T) {
	clk := newClock()
	tb := newLeaseTable(clk.now, 20*time.Second)
	tb.own("A", resource{resOpenVPN, "s1"})
	tb.own("B", resource{resKillSwitch, ""})
	tb.own("", resource{resHosts, ""}) // unleased: never expires

	for i := 0; i < 5; i++ { // B renews every 5s, A never
		clk.add(5 * time.Second)
		tb.touch("B")
		got := tb.expire()
		if i < 3 && len(got) != 0 {
			t.Fatalf("step %d: expired too early: %v", i, got)
		}
		if i == 3 {
			if want := []resource{{resOpenVPN, "s1"}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("at 20s got %v, want %v", got, want)
			}
		}
	}
	if tb.count() != 1 {
		t.Fatalf("live leases = %d, want 1 (B)", tb.count())
	}
	clk.add(20 * time.Second)
	if got := tb.expire(); !reflect.DeepEqual(got, []resource{{resKillSwitch, ""}}) {
		t.Fatalf("B expiry got %v", got)
	}
}

func TestLeaseTable_ReleaseAndTakeover(t *testing.T) {
	clk := newClock()
	tb := newLeaseTable(clk.now, 20*time.Second)
	tb.own("A", resource{resKillSwitch, ""})
	tb.own("B", resource{resKillSwitch, ""}) // re-enabled by B: B owns it now
	tb.own("A", resource{resDNS, "wg0"})
	tb.release(resource{resDNS, "wg0"}) // restored explicitly
	clk.add(10 * time.Second)
	tb.touch("B")
	clk.add(10 * time.Second)
	if got := tb.expire(); len(got) != 0 {
		t.Fatalf("A owned nothing left, got %v", got)
	}
	tb.own("", resource{resKillSwitch, ""}) // elevated caller took it over
	clk.add(time.Minute)
	if got := tb.expire(); len(got) != 0 {
		t.Fatalf("unleased resource expired: %v", got)
	}
}

func TestLocal_ReapExpiredLeases_OrderAndScope(t *testing.T) {
	clk := newClock()
	l := NewLocal(LocalOptions{StateDir: t.TempDir(), Now: clk.now})
	var released []string
	l.releaseFn = func(_ context.Context, r resource) error {
		released = append(released, r.String())
		return nil
	}
	ctxA := withLease(context.Background(), "uid/A")
	ctxB := withLease(context.Background(), "uid/B")
	l.leases.own(l.leaseOf(ctxA), resource{resKillSwitch, ""})
	l.leases.own(l.leaseOf(ctxA), resource{resHosts, ""})
	l.leases.own(l.leaseOf(ctxA), resource{resDNS, "tun0"})
	l.leases.own(l.leaseOf(ctxA), resource{resOpenVPN, "s1"})
	l.leases.own(l.leaseOf(ctxB), resource{resOpenVPN, "s2"})

	clk.add(15 * time.Second)
	if err := l.RenewLease(ctxB); err != nil {
		t.Fatal(err)
	}
	if rel := l.ReapExpiredLeases(context.Background()); len(rel) != 0 {
		t.Fatalf("reaped before TTL: %v", rel)
	}
	clk.add(6 * time.Second) // A at 21s, B at 6s since renewal
	rel := l.ReapExpiredLeases(context.Background())
	want := []string{"openvpn:s1", "dns:tun0", "hosts", "killswitch"}
	if !reflect.DeepEqual(released, want) {
		t.Fatalf("released %v, want %v (tunnel first, kill switch last)", released, want)
	}
	if len(rel) != len(want) || rel[0].Resource != "openvpn:s1" {
		t.Fatalf("report = %+v", rel)
	}
	released = nil
	clk.add(20 * time.Second)
	l.ReapExpiredLeases(context.Background())
	if !reflect.DeepEqual(released, []string{"openvpn:s2"}) {
		t.Fatalf("B expiry released %v", released)
	}
}

func TestDispatch_ScopesLeaseToCaller(t *testing.T) {
	f := &fakeOps{}
	ctx := WithCaller(context.Background(), Caller{ID: "S-1-5-21-1", Name: "u"})
	lease := strings.Repeat("ab", 16)
	resp := Dispatch(ctx, f, Request{V: ProtocolVersion, Op: OpLeaseRenew, Lease: lease})
	if !resp.OK || f.last != OpLeaseRenew || f.arg != "S-1-5-21-1/"+lease {
		t.Fatalf("resp=%+v last=%q arg=%v", resp, f.last, f.arg)
	}
	resp = Dispatch(ctx, f, Request{V: ProtocolVersion, Op: OpPing, Lease: "../x"})
	if resp.OK || resp.Code != errCodeInvalid {
		t.Fatalf("bad lease accepted: %+v", resp)
	}
}

// renewCounter counts lease renewals reaching the helper side.
type renewCounter struct {
	fakeOps
	n     atomic.Int64
	lease atomic.Value
}

func (r *renewCounter) RenewLease(ctx context.Context) error {
	r.n.Add(1)
	r.lease.Store(leaseFrom(ctx))
	return nil
}

func TestClient_RenewsOnlyAfterOwningState(t *testing.T) {
	r := &renewCounter{}
	c := pipeClient(r)
	c.RenewEvery = 10 * time.Millisecond
	defer c.Close()
	ctx := context.Background()

	if _, err := c.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if r.n.Load() != 0 {
		t.Fatalf("renewed without owning state: %d", r.n.Load())
	}
	if err := c.KillSwitchEnable(ctx, KillSwitchRequest{}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for r.n.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if r.n.Load() < 2 {
		t.Fatalf("lease not renewed (renewals=%d)", r.n.Load())
	}
	if got, _ := r.lease.Load().(string); !strings.HasSuffix(got, "/"+c.lease) {
		t.Fatalf("renewed lease %q, client lease %q", got, c.lease)
	}
	c.Close()
	time.Sleep(30 * time.Millisecond)
	n := r.n.Load()
	time.Sleep(60 * time.Millisecond)
	if r.n.Load() > n+1 {
		t.Fatalf("renewal continued after Close")
	}
}
