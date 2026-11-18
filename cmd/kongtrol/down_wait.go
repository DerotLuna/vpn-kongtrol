package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// downPollInterval is how often `down` re-reads a tunnel the user must
// disconnect in the provider app.
var downPollInterval = time.Second

// localTunnelState reads profile name's state from this process's adapter
// (no daemon running).
func localTunnelState(name string) (monitor.TunnelMetrics, bool) {
	a, ok := adapters[name]
	if !ok {
		return monitor.TunnelMetrics{}, false
	}
	m := monitor.TunnelMetrics{Name: name, Status: a.Status().Normalize()}
	if r, ok := a.(vpn.AttachReporter); ok {
		m.Attach = r.AttachState()
	}
	if r, ok := a.(vpn.HealthReporter); ok {
		m.Health = r.TunnelHealth()
	}
	return m, true
}

// manualDownTimeout is the profile's wait_timeout (default 5m).
func manualDownTimeout(name string) time.Duration {
	if cfg != nil {
		if d, err := time.ParseDuration(cfg.VPNs[name].WaitTimeout); err == nil && d > 0 {
			return d
		}
	}
	return 5 * time.Minute
}

// waitManualDown waits until the user has disconnected profile name in the
// provider app (its status is no longer connected), showing the attach
// message on spin (or printing it once when spin is nil, e.g. --all).
// Ctrl+C or the profile's wait_timeout stop waiting: the tunnel stays up
// and the returned error says so (wrapping vpn.ErrManualDisconnect).
func waitManualDown(ctx context.Context, name string, read func(string) (monitor.TunnelMetrics, bool), spin *spinner) error {
	deadline := time.Now().Add(manualDownTimeout(name))
	printed := false
	tick := time.NewTicker(downPollInterval)
	defer tick.Stop()
	for {
		st, ok := read(name)
		if ok && !downStillWaiting(st) {
			return nil
		}
		if ok {
			if spin != nil {
				if l := attachSpinnerLabel(name, st.Attach, "attach.hint.stop_waiting"); l != "" {
					spin.SetLabel(l)
				}
			} else if !printed {
				if line := attachLine(st.Attach); line != "" {
					fmt.Println(tuiInfo(paintBold(cBright, name) + "  " + paint(cDim, line)))
					printed = true
				}
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s: %w", cf("cli.attach.disconnect_timeout", name), vpn.ErrManualDisconnect)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", cf("cli.attach.disconnect_timeout", name), vpn.ErrManualDisconnect)
		case <-tick.C:
		}
	}
}

// downStillWaiting reports whether a tunnel the user must close in the app
// is still up.
func downStillWaiting(st monitor.TunnelMetrics) bool {
	return st.Status.Normalize() == vpn.StatusConnected
}
