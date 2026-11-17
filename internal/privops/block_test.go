package privops

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
)

func prefixes(ss ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParsePrefix(s)
	}
	return out
}

func TestNormalizeBlock(t *testing.T) {
	req, err := NormalizeBlock(BlockRequest{Destinations: prefixes("10.1.2.3/16", "1.1.1.1/32", "10.1.0.0/16", "2001:db8::1/32")})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, p := range req.Destinations {
		got = append(got, p.String())
	}
	if strings.Join(got, " ") != "1.1.1.1/32 10.1.0.0/16 2001:db8::/32" {
		t.Fatalf("got %v", got)
	}
	for _, bad := range []string{"0.0.0.0/0", "127.0.0.1/32", "224.0.0.0/8", "::/0", "ff02::1/128"} {
		if _, err := NormalizeBlock(BlockRequest{Destinations: prefixes(bad)}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s accepted", bad)
		}
	}
	if _, err := NormalizeBlock(BlockRequest{Destinations: []netip.Prefix{{}}}); !errors.Is(err, ErrInvalidRequest) {
		t.Error("invalid prefix accepted")
	}
	big := make([]netip.Prefix, MaxBlockPrefixes+1)
	for i := range big {
		big[i] = netip.PrefixFrom(netip.AddrFrom4([4]byte{10, byte(i >> 8), byte(i), 0}), 24)
	}
	if _, err := NormalizeBlock(BlockRequest{Destinations: big}); !errors.Is(err, ErrInvalidRequest) {
		t.Error("oversized set accepted")
	}
}

func TestBlockRoundTripAndLease(t *testing.T) {
	f := &fakeOps{}
	c := pipeClient(f)
	defer c.Close()
	req := BlockRequest{Destinations: prefixes("203.0.113.0/24")}
	if err := c.SetBlockedDestinations(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if f.last != OpBlockSet || f.arg.(BlockRequest).Destinations[0].String() != "203.0.113.0/24" {
		t.Fatalf("got %s %+v", f.last, f.arg)
	}
	if !leasedOp(OpBlockSet) {
		t.Fatal("block.set must be leased (fail-open on daemon crash)")
	}
}

func TestBlockOldHelperIsOutdated(t *testing.T) {
	// An older helper answers "unknown operation" for block.set.
	f := &fakeOps{err: invalid("unknown operation")}
	cl := pipeClient(f)
	defer cl.Close()
	if err := cl.SetBlockedDestinations(context.Background(), BlockRequest{}); !errors.Is(err, ErrHelperOutdated) {
		t.Fatalf("err = %v, want ErrHelperOutdated", err)
	}
}

func TestReapReleasesBlockBeforeKillSwitch(t *testing.T) {
	clk := newClock()
	l := NewLocal(LocalOptions{StateDir: t.TempDir(), Now: clk.now})
	var released []string
	l.releaseFn = func(_ context.Context, r resource) error {
		released = append(released, r.String())
		return nil
	}
	ctx := withLease(context.Background(), "uid/A")
	l.leases.own(l.leaseOf(ctx), resource{resKillSwitch, ""})
	l.leases.own(l.leaseOf(ctx), resource{resBlock, ""})
	clk.add(LeaseTTL + 1)
	l.ReapExpiredLeases(context.Background())
	if strings.Join(released, ",") != "block,killswitch" {
		t.Fatalf("released %v", released)
	}
}

func TestPFBlockRules(t *testing.T) {
	got := pfBlockRules(prefixes("10.0.0.0/8", "2001:db8::/32"))
	if !strings.Contains(got, "table <kongtrol_block> const { 10.0.0.0/8 2001:db8::/32 }") ||
		!strings.Contains(got, "block return out quick to <kongtrol_block>") {
		t.Fatalf("rules:\n%s", got)
	}
	if n := len(chunkPrefixes(make([]netip.Prefix, 401), 200, ",")); n != 3 {
		t.Fatalf("chunks = %d", n)
	}
}
