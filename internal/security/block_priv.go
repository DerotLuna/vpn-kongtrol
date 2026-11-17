package security

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// DestinationBlocker blocks outbound traffic to a set of destinations on
// every interface (per-destination kill switch, policies with
// on_tunnel_down: block). Set replaces the whole set; an empty set removes
// the block. The helper leases it: if the daemon dies it is removed.
type DestinationBlocker interface {
	Set(dsts []netip.Prefix) error
	Blocked() []netip.Prefix
}

type privBlocker struct {
	mu      sync.Mutex
	current []netip.Prefix
	applied bool // a non-empty set is in place
}

// NewDestinationBlocker returns the blocker backed by privops.
func NewDestinationBlocker() DestinationBlocker { return &privBlocker{} }

// Set sends dsts to the helper unless it equals what is in place.
func (b *privBlocker) Set(dsts []netip.Prefix) error {
	dsts = slices.Clone(dsts)
	slices.SortFunc(dsts, comparePrefix)
	dsts = slices.Compact(dsts)
	b.mu.Lock()
	defer b.mu.Unlock()
	if slices.Equal(dsts, b.current) && (b.applied || len(dsts) == 0) {
		return nil
	}
	err := withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.SetBlockedDestinations(ctx, privops.BlockRequest{Destinations: dsts})
	})
	if err != nil {
		if len(dsts) == 0 && !b.applied {
			return nil // nothing of ours is in place
		}
		return fmt.Errorf("block destinations: %w", err)
	}
	b.current, b.applied = dsts, len(dsts) > 0
	return nil
}

func (b *privBlocker) Blocked() []netip.Prefix {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.current)
}

func comparePrefix(a, c netip.Prefix) int {
	if x := a.Addr().Compare(c.Addr()); x != 0 {
		return x
	}
	return a.Bits() - c.Bits()
}
