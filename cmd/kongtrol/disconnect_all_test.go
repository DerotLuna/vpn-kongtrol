package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// TestDisconnectAllConcurrently_RunsInParallel proves profiles are
// disconnected concurrently, not one at a time: with N profiles that each
// take `delay` to disconnect, sequential execution would take N*delay;
// this must finish in roughly one delay's worth of time regardless of N.
func TestDisconnectAllConcurrently_RunsInParallel(t *testing.T) {
	const n = 5
	const delay = 200 * time.Millisecond

	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("profile-%d", i)
	}

	var inFlight, maxInFlight atomic.Int32
	disconnect := func(ctx context.Context, name string) error {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			max := maxInFlight.Load()
			if cur <= max || maxInFlight.CompareAndSwap(max, cur) {
				break
			}
		}
		time.Sleep(delay)
		return nil
	}

	start := time.Now()
	disconnectAllConcurrently(context.Background(), names, disconnect)
	elapsed := time.Since(start)

	if elapsed >= n*delay {
		t.Fatalf("took %v — expected roughly one delay (%v), not sequential (%v)", elapsed, delay, n*delay)
	}
	if got := maxInFlight.Load(); got < 2 {
		t.Fatalf("max concurrent disconnects observed = %d, want > 1 (evidence of true parallelism, not accidental fast sequential execution)", got)
	}
	t.Logf("elapsed=%v maxConcurrent=%d", elapsed, maxInFlight.Load())
}

// TestDisconnectAllConcurrently_ContinuesPastIndividualErrors preserves the
// original --all semantics: every profile is attempted regardless of
// another profile's failure, and a failure doesn't propagate as a command
// error (the original code used `_ = disconnectWithSpinner(name)`).
func TestDisconnectAllConcurrently_ContinuesPastIndividualErrors(t *testing.T) {
	names := []string{"good-1", "bad", "good-2"}
	var attempted atomic.Int32
	disconnect := func(ctx context.Context, name string) error {
		attempted.Add(1)
		if name == "bad" {
			return fmt.Errorf("simulated failure")
		}
		return nil
	}

	// Must not panic and must attempt every profile even though one errors.
	disconnectAllConcurrently(context.Background(), names, disconnect)

	if got := attempted.Load(); got != int32(len(names)) {
		t.Fatalf("attempted %d profiles, want %d — one failure must not skip the others", got, len(names))
	}
}
