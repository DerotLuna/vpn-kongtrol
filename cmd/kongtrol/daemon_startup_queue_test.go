package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestStartupQueueCancelBeforeStart(t *testing.T) {
	var mu sync.Mutex
	states := map[string]bool{}
	q := newStartupQueue(func(name string, pending, _ bool) {
		mu.Lock()
		states[name] = pending
		mu.Unlock()
	})
	q.add(t.Context(), "work", true, true)
	if err := q.cancel(t.Context(), "work"); err != nil {
		t.Fatal(err)
	}
	called := false
	q.run("work", func(context.Context) { called = true })
	if called || states["work"] {
		t.Fatal("cancelled queued connect started or remained pending")
	}
}

func TestStartupQueueCancelWaitsForActiveConnect(t *testing.T) {
	q := newStartupQueue(func(string, bool, bool) {})
	q.add(t.Context(), "work", true, true)
	entered, release := make(chan struct{}), make(chan struct{})
	q.goRun("work", func(ctx context.Context) {
		close(entered)
		<-ctx.Done()
		<-release
	})
	<-entered
	done := make(chan error, 1)
	go func() { done <- q.cancel(t.Context(), "work") }()
	select {
	case err := <-done:
		t.Fatalf("cancel returned before the connect stopped: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestStartupQueueDisableRestoreDoesNotCancelRestart(t *testing.T) {
	q := newStartupQueue(func(string, bool, bool) {})
	q.add(t.Context(), "restore", true, true)
	q.add(t.Context(), "restart", false, true)
	q.stopRestores(false)
	ran := false
	q.run("restore", func(context.Context) { t.Error("disabled restore ran") })
	q.run("restart", func(context.Context) { ran = true })
	if !ran {
		t.Fatal("explicit restart did not run")
	}
}

func TestStartupQueueScheduledBeforeWorkerRuns(t *testing.T) {
	q := newStartupQueue(func(string, bool, bool) {})
	q.add(t.Context(), "work", true, true)
	work := q.begin("work", func(context.Context) { t.Error("cancelled worker ran") })
	q.stop("work")
	work()
	if q.hasRestores() {
		t.Fatal("cancelled restore remained pending")
	}
}
