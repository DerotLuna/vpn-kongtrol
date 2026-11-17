package openvpn

import (
	"context"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// An openvpn killed outside Kongtrol (the kill-switch test that exposed
// this) must not leave the adapter answering "already connected": the
// watchdog's reconnect has to start a fresh session.
func TestAdapter_ReconnectAfterProcessExit(t *testing.T) {
	m := newFakeMgmt(t, "s3cret")
	h := &fakeHelper{addr: m.ln.Addr().String()}
	useFakeHelper(t, h, nil)

	a := &Adapter{}
	cfg := vpn.AdapterConfig{ConfigPath: "office.ovpn"}
	if err := a.Connect(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if st := a.Status(); st != vpn.StatusConnected {
		t.Fatalf("status after connect = %s", st)
	}

	// openvpn dies on the helper side.
	h.mu.Lock()
	h.running = false
	h.mu.Unlock()
	deadline := time.Now().Add(2 * time.Second)
	for a.Status() != vpn.StatusError && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := a.Status(); st != vpn.StatusError {
		t.Fatalf("dead process reported as %s, want error", st)
	}

	h.mu.Lock()
	h.stopped = false
	h.mu.Unlock()
	if err := a.Connect(context.Background(), cfg); err != nil {
		t.Fatalf("reconnect after exit: %v", err)
	}
	if st := a.Status(); st != vpn.StatusConnected {
		t.Fatalf("status after reconnect = %s", st)
	}
	h.mu.Lock()
	released := h.stopped
	h.mu.Unlock()
	if !released {
		t.Fatal("the dead helper session was not stopped before starting a new one")
	}
	if err := a.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// Disconnect of a directly spawned openvpn that already exited succeeds
// and leaves the adapter disconnected (nothing to signal).
func TestAdapter_DisconnectAfterDirectExit(t *testing.T) {
	p := &process{log: &logState{}, exited: make(chan struct{})}
	p.markExited(nil)
	a := &Adapter{status: vpn.StatusConnected, proc: p}
	if st := a.Status(); st != vpn.StatusError {
		t.Fatalf("status = %s, want error", st)
	}
	if err := a.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if st := a.Status(); st != vpn.StatusDisconnected || a.proc != nil {
		t.Fatalf("status = %s proc = %v, want disconnected and no process", st, a.proc)
	}
}
