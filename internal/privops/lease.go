package privops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Leases tie what the helper does on behalf of a client to that client
// staying alive.
//
// Every Client process picks a random lease id and sends it with each
// request. Requests that leave state behind (an OpenVPN process, the kill
// switch, forced DNS, the hosts block) record the lease as the owner of
// that state. The client renews its lease every LeaseRenewEvery; a lease
// not renewed for LeaseTTL expires and the helper undoes everything it
// owns. So a daemon that crashes or is killed never leaves an unmanaged
// tunnel, a firewall that blocks the network, or resolvers pointing at a
// dead tunnel.
//
// Expiry restores state (fail-open): the kill switch is removed rather
// than left blocking. A crashed daemon can no longer watch, reconnect or
// switch the kill switch off, so keeping the block would leave the user
// with no network and no Kongtrol to fix it — and the Windows firewall
// rules and static DNS persist across reboots. Leaving traffic unprotected
// for a moment is the lesser harm; the daemon re-applies the kill switch
// when it comes back and reconnects.
//
// Not leased, on purpose: WireGuard tunnels (an OS service that runs on
// its own and that a restarted daemon adopts; `kongtrol down` removes it)
// and routes (they live and die with the tunnel interface).
//
// Requests without a lease (an elevated process using Local directly)
// own nothing that expires.
const (
	LeaseTTL        = 20 * time.Second
	LeaseRenewEvery = 5 * time.Second
)

type leaseKey struct{}

// withLease attaches the lease id (already scoped to the caller) to ctx.
func withLease(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, leaseKey{}, id)
}

// leaseFrom returns the lease attached to ctx, or "".
func leaseFrom(ctx context.Context) string {
	id, _ := ctx.Value(leaseKey{}).(string)
	return id
}

// ValidateLeaseID checks a client lease id (32 lowercase hex characters).
func ValidateLeaseID(id string) error {
	if !sessionIDRE.MatchString(id) {
		return invalid("bad lease id")
	}
	return nil
}

// resourceKind is a kind of state a lease can own.
type resourceKind int

const (
	resOpenVPN    resourceKind = iota // id = session id
	resKillSwitch                     // single, id = ""
	resDNS                            // id = interface
	resHosts                          // single, id = ""
	resBlock                          // single, id = "" (blocked destinations)
	resWGAppRoute                     // id = WireGuard interface
)

func (k resourceKind) String() string {
	switch k {
	case resOpenVPN:
		return "openvpn"
	case resKillSwitch:
		return "killswitch"
	case resDNS:
		return "dns"
	case resHosts:
		return "hosts"
	case resBlock:
		return "block"
	case resWGAppRoute:
		return "app-route"
	}
	return "unknown"
}

// resource is one piece of state owned by a lease.
type resource struct {
	kind resourceKind
	id   string
}

func (r resource) String() string {
	if r.id == "" {
		return r.kind.String()
	}
	return r.kind.String() + ":" + r.id
}

// leaseTable tracks lease deadlines and which lease owns each resource.
// It has its own lock so renewals never wait behind a slow system call.
type leaseTable struct {
	mu     sync.Mutex
	now    func() time.Time
	ttl    time.Duration
	expiry map[string]time.Time // lease → deadline
	owner  map[resource]string  // resource → lease
}

func newLeaseTable(now func() time.Time, ttl time.Duration) *leaseTable {
	if now == nil {
		now = time.Now
	}
	if ttl <= 0 {
		ttl = LeaseTTL
	}
	return &leaseTable{now: now, ttl: ttl, expiry: map[string]time.Time{}, owner: map[resource]string{}}
}

// touch renews lease (no-op for "").
func (t *leaseTable) touch(lease string) {
	if lease == "" {
		return
	}
	t.mu.Lock()
	t.expiry[lease] = t.now().Add(t.ttl)
	t.mu.Unlock()
}

// own records lease as r's owner. An empty lease makes r unleased: it
// never expires (an elevated caller took it over).
func (t *leaseTable) own(lease string, r resource) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if lease == "" {
		delete(t.owner, r)
		return
	}
	t.owner[r] = lease
	t.expiry[lease] = t.now().Add(t.ttl)
}

// release forgets r (it was undone by an explicit request).
func (t *leaseTable) release(r resource) {
	t.mu.Lock()
	delete(t.owner, r)
	t.mu.Unlock()
}

// expire removes every lease past its deadline and returns the resources
// they owned, sorted for a stable cleanup order.
func (t *leaseTable) expire() []resource {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	dead := map[string]bool{}
	for l, d := range t.expiry {
		if !now.Before(d) {
			dead[l] = true
			delete(t.expiry, l)
		}
	}
	var out []resource
	for r, l := range t.owner {
		if dead[l] {
			out = append(out, r)
			delete(t.owner, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].kind != out[j].kind {
			return out[i].kind < out[j].kind
		}
		return out[i].id < out[j].id
	})
	return out
}

// leases reports the live lease count (tests, status).
func (t *leaseTable) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.expiry)
}

// ── Local integration ───────────────────────────────────────────────────

// leaseOf renews and returns the lease of the request in ctx.
func (l *Local) leaseOf(ctx context.Context) string {
	id := leaseFrom(ctx)
	l.leases.touch(id)
	return id
}

// RenewLease keeps the caller's lease alive.
func (l *Local) RenewLease(ctx context.Context) error {
	l.leaseOf(ctx)
	return nil
}

// ExpiredRelease is one resource undone because its lease expired.
type ExpiredRelease struct {
	Resource string `json:"resource"`
	Error    string `json:"error,omitempty"`
}

// ReapExpiredLeases undoes everything owned by expired leases, in a safe
// order: OpenVPN processes first, then forced DNS and the hosts block,
// the kill switch last (so traffic is never released while a tunnel is
// still being torn down).
func (l *Local) ReapExpiredLeases(ctx context.Context) []ExpiredRelease {
	res := l.leases.expire()
	sort.SliceStable(res, func(i, j int) bool { return reapOrder(res[i].kind) < reapOrder(res[j].kind) })
	var out []ExpiredRelease
	for _, r := range res {
		e := ExpiredRelease{Resource: r.String()}
		if err := l.releaseFn(ctx, r); err != nil {
			e.Error = err.Error()
		}
		out = append(out, e)
	}
	return out
}

func reapOrder(k resourceKind) int {
	switch k {
	case resOpenVPN:
		return 0
	case resDNS:
		return 1
	case resHosts:
		return 2
	case resBlock:
		return 3
	}
	return 4 // kill switch
}

// RunLeaseReaper calls ReapExpiredLeases every interval until ctx ends,
// passing each non-empty result to report.
func (l *Local) RunLeaseReaper(ctx context.Context, every time.Duration, report func([]ExpiredRelease)) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if rel := l.ReapExpiredLeases(ctx); len(rel) > 0 && report != nil {
				report(rel)
			}
		}
	}
}

// releaseResource undoes one resource as the helper itself (no owner
// check: the owner is gone).
func (l *Local) releaseResource(ctx context.Context, r resource) error {
	switch r.kind {
	case resOpenVPN:
		l.mu.Lock()
		s, ok := l.sessions[r.id]
		l.mu.Unlock()
		if !ok {
			return nil
		}
		l.stopSession(s)
		return nil
	case resKillSwitch:
		return l.KillSwitchDisable(ctx)
	case resDNS:
		return l.RestoreDNS(ctx, r.id)
	case resHosts:
		return l.WriteHostsBlock(ctx, nil)
	case resBlock:
		return l.SetBlockedDestinations(ctx, BlockRequest{})
	case resWGAppRoute:
		l.mu.Lock()
		defer l.mu.Unlock()
		// Once the app lease expires, a running interface is no longer a
		// usable app tunnel. Tear it down so the next daemon can reconnect.
		if _, ok := l.wgAppRouting[r.id]; !ok {
			return nil
		}
		copyPath := filepath.Join(l.wgDir(), r.id+".conf")
		err := wgDownFn(ctx, r.id, copyPath)
		if err == nil {
			_ = os.Remove(copyPath)
		}
		return errors.Join(err, l.clearWGAppRouting(r.id))
	}
	return nil
}
