package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// TestConnectViaDaemon_WaitingUserSuspendsTimeout: while the provider
// adapter waits for the user in its app, `up` keeps waiting past its own
// connect timeout and reports every phase to the spinner callback.
func TestConnectViaDaemon_WaitingUserSuspendsTimeout(t *testing.T) {
	stubUp(t)
	oldTimeout := upConnectTimeout
	upConnectTimeout = 20 * time.Millisecond
	t.Cleanup(func() { upConnectTimeout = oldTimeout })
	daemonConnectFn = func(string, string) (bool, error) { return false, nil }

	var polls atomic.Int32
	waiting := &vpn.AttachState{Phase: vpn.PhaseWaitingUser, App: "Surfshark"}
	daemonTunnelStatesFn = func(string) (map[string]monitor.TunnelMetrics, error) {
		if polls.Add(1) < 15 { // ~75ms, well past upConnectTimeout
			return map[string]monitor.TunnelMetrics{"a": {Name: "a", Status: vpn.StatusConnecting, Attach: waiting}}, nil
		}
		return map[string]monitor.TunnelMetrics{"a": {Name: "a", Status: vpn.StatusConnected,
			Attach: &vpn.AttachState{Phase: vpn.PhaseUser, App: "Surfshark"}}}, nil
	}
	var phases []vpn.AttachPhase
	_, err := connectViaDaemonProgress(context.Background(), "http://x", "a", func(m monitor.TunnelMetrics) {
		if m.Attach != nil && (len(phases) == 0 || phases[len(phases)-1] != m.Attach.Phase) {
			phases = append(phases, m.Attach.Phase)
		}
	})
	if err != nil {
		t.Fatalf("err = %v, want connected after the user connects", err)
	}
	if len(phases) != 2 || phases[0] != vpn.PhaseWaitingUser || phases[1] != vpn.PhaseUser {
		t.Errorf("phases = %v", phases)
	}
}
