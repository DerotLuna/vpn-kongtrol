package main

import (
	"context"
	"sync"
)

// startupQueue owns connects planned before the API starts. Removing a
// queued entry and starting it are serialized with a user cancellation.
type startupQueue struct {
	mu      sync.Mutex
	entries map[string]*startupEntry
	update  func(string, bool, bool)
}

type startupEntry struct {
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
	restore bool
}

func newStartupQueue(update func(string, bool, bool)) *startupQueue {
	return &startupQueue{entries: make(map[string]*startupEntry), update: update}
}

func (q *startupQueue) add(ctx context.Context, name string, restore, reconnect bool) {
	child, cancel := context.WithCancel(ctx)
	q.mu.Lock()
	if _, exists := q.entries[name]; exists {
		q.mu.Unlock()
		cancel()
		return
	}
	q.entries[name] = &startupEntry{ctx: child, cancel: cancel, done: make(chan struct{}), restore: restore}
	q.update(name, true, reconnect)
	q.mu.Unlock()
}

func (q *startupQueue) run(name string, fn func(context.Context)) {
	if work := q.begin(name, fn); work != nil {
		work()
	}
}

func (q *startupQueue) goRun(name string, fn func(context.Context)) {
	if work := q.begin(name, fn); work != nil {
		go work()
	}
}

func (q *startupQueue) begin(name string, fn func(context.Context)) func() {
	q.mu.Lock()
	e := q.entries[name]
	if e == nil || e.started {
		q.mu.Unlock()
		return nil
	}
	e.started = true
	q.mu.Unlock()
	return func() {
		defer func() {
			e.cancel()
			q.mu.Lock()
			if q.entries[name] == e {
				delete(q.entries, name)
				q.update(name, false, false)
			}
			close(e.done)
			q.mu.Unlock()
		}()
		if e.ctx.Err() == nil {
			fn(e.ctx)
		}
	}
}

// stop prevents a queued connect from starting, or cancels an active one.
func (q *startupQueue) stop(name string) <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	e := q.entries[name]
	if e == nil {
		return nil
	}
	e.cancel()
	if !e.started {
		delete(q.entries, name)
		q.update(name, false, false)
		close(e.done)
	}
	return e.done
}

func (q *startupQueue) cancel(ctx context.Context, name string) error {
	done := q.stop(name)
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *startupQueue) stopRestores(onlyQueued bool) {
	q.mu.Lock()
	names := make([]string, 0, len(q.entries))
	for name, e := range q.entries {
		if e.restore && (!onlyQueued || !e.started) {
			names = append(names, name)
		}
	}
	q.mu.Unlock()
	for _, name := range names {
		q.stop(name)
	}
}

func (q *startupQueue) stopAll(ctx context.Context) {
	q.mu.Lock()
	names := make([]string, 0, len(q.entries))
	for name := range q.entries {
		names = append(names, name)
	}
	q.mu.Unlock()
	for _, name := range names {
		q.stop(name)
	}
	for _, name := range names {
		_ = q.cancel(ctx, name)
	}
}

func (q *startupQueue) hasRestores() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, e := range q.entries {
		if e.restore {
			return true
		}
	}
	return false
}
