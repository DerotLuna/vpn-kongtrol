package monitor

import (
	"context"
	"net"
	"testing"
	"time"
)

func instantClock(count *int) func(time.Duration) <-chan time.Time {
	return func(time.Duration) <-chan time.Time {
		*count++
		ch := make(chan time.Time, 1)
		ch <- time.Time{}
		return ch
	}
}

func TestNetworkWait(t *testing.T) {
	polls := 0
	w := NetworkWait{Ready: func() bool { return true }, Interval: 2 * time.Second, Timeout: 90 * time.Second, After: instantClock(&polls)}
	if !w.Wait(context.Background()) || polls != 0 {
		t.Fatalf("ready network waited %d polls", polls)
	}

	calls := 0
	w.Ready = func() bool { calls++; return calls == 5 }
	if !w.Wait(context.Background()) || polls != 4 {
		t.Fatalf("ready after 4 polls: polls=%d", polls)
	}

	polls = 0
	w.Ready = func() bool { return false }
	if w.Wait(context.Background()) || polls != 45 {
		t.Fatalf("no network: polls=%d, want 45 (90 s / 2 s)", polls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.After = func(time.Duration) <-chan time.Time { return make(chan time.Time) }
	if w.Wait(ctx) {
		t.Fatal("a cancelled wait reported a network")
	}
}

func TestWaitNetworkChange(t *testing.T) {
	prints := []string{"", "", "wifi=10.0.0.2", "wifi=10.0.0.2", "eth=10.0.1.2"}
	i := 0
	fp := func() string { v := prints[min(i, len(prints)-1)]; i++; return v }
	readyCalls := 0
	ready := func() bool { readyCalls++; return readyCalls == 2 } // the first change has no route yet
	polls := 0
	if !WaitNetworkChange(context.Background(), fp, ready, 5*time.Second, instantClock(&polls)) {
		t.Fatal("change not seen")
	}
	if readyCalls != 2 || polls != 4 {
		t.Fatalf("ready checked %d times after %d polls; want only on changes", readyCalls, polls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if WaitNetworkChange(ctx, func() string { return "" }, ready, time.Second, func(time.Duration) <-chan time.Time { return make(chan time.Time) }) {
		t.Fatal("cancelled wait reported a change")
	}
}

func TestUsableSource(t *testing.T) {
	for ip, want := range map[string]bool{
		"192.168.1.5": true, "10.0.0.2": true, "2001:db8::5": true,
		"169.254.10.1": false, "fe80::1": false, "127.0.0.1": false, "0.0.0.0": false,
	} {
		if got := UsableSource(net.ParseIP(ip)); got != want {
			t.Errorf("UsableSource(%s) = %v, want %v", ip, got, want)
		}
	}
	if UsableSource(nil) {
		t.Error("nil is usable")
	}
}

// Smoke test on the build machine: the probe must not fail or hang, and
// excluding every interface means no network.
func TestNetworkReadySmoke(t *testing.T) {
	t.Logf("NetworkReady on this machine: %v", NetworkReady(nil))
	if NetworkReady(func(string) bool { return true }) {
		t.Fatal("ready although every interface is one of our tunnels")
	}
}
