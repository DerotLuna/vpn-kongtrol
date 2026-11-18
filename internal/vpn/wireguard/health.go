package wireguard

import (
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// WireGuard has no connection state: the only liveness signal is the
// peer's latest handshake (whitepaper §6.1; wireguard-go
// device/constants.go). A session renews every REKEY_AFTER_TIME (120 s)
// while traffic flows and its keys are unusable after REJECT_AFTER_TIME
// (180 s). Handshakes only happen when something is sent: traffic or
// PersistentKeepalive. See ADR-0030 and docs/research/2026-11-18-wireguard-liveness.md.
const (
	// staleHandshake: a handshake older than REJECT_AFTER_TIME + margin
	// means no live session.
	staleHandshake = 180*time.Second + 20*time.Second
	// handshakeGrace: time after connect (or after Kongtrol starts
	// watching) before the tunnel can be judged; handshake retries run
	// every REKEY_TIMEOUT (5 s).
	handshakeGrace = 30 * time.Second
	// stallWindow: tx kept growing with rx flat for this long (three
	// REKEY_TIMEOUT retries) → the server does not answer.
	stallWindow = 15 * time.Second
)

// sample is a tunnel's counters, aggregated over its peers.
type sample struct {
	handshake time.Time // latest across peers (zero = never)
	rx, tx    uint64
	keepalive bool // any peer has PersistentKeepalive
}

func aggregate(st privops.WGStats) sample {
	var s sample
	for _, p := range st.Peers {
		if p.LatestHandshake > 0 {
			if hs := time.Unix(p.LatestHandshake, 0); hs.After(s.handshake) {
				s.handshake = hs
			}
		}
		s.rx += p.RxBytes
		s.tx += p.TxBytes
		if p.PersistentKeepalive > 0 {
			s.keepalive = true
		}
	}
	return s
}

// healthTracker turns successive counter samples into a vpn.TunnelHealth.
// Rule (ADR-0030):
//   - handshake younger than staleHandshake → ok;
//   - otherwise, after handshakeGrace, no_response when PersistentKeepalive
//     is set (it forces handshakes), or when tx grew while rx stayed flat
//     for stallWindow, or when it never handshaked but sent something;
//   - otherwise idle (unused tunnel without keepalive: nothing to judge).
//
// Not safe for concurrent use; the adapter serializes calls.
type healthTracker struct {
	start      time.Time // connect or first observation
	have       bool
	prevRx     uint64
	prevTx     uint64
	stallSince time.Time
	stallRx    uint64
	cur        vpn.TunnelHealth
}

// reset starts judging a fresh tunnel at now.
func (t *healthTracker) reset(now time.Time) {
	*t = healthTracker{start: now}
}

// observe records one read of the counters (err when they could not be
// read) and returns the resulting health.
func (t *healthTracker) observe(now time.Time, st privops.WGStats, err error) vpn.TunnelHealth {
	if t.start.IsZero() {
		t.start = now
	}
	if err != nil {
		key, _ := privops.GuidanceKey(err)
		return t.set(now, vpn.TunnelHealth{State: vpn.HealthUnknown, ReasonKey: key})
	}
	s := aggregate(st)
	if t.have && (s.rx < t.prevRx || s.tx < t.prevTx) {
		// Counters went back: the tunnel was re-created.
		t.stallSince = time.Time{}
	}
	fresh := !s.handshake.IsZero() && now.Sub(s.handshake) <= staleHandshake

	state := vpn.HealthOK
	if fresh {
		t.stallSince = time.Time{}
	} else {
		switch {
		case !t.stallSince.IsZero() && s.rx > t.stallRx:
			t.stallSince = time.Time{} // the server answered
		case t.stallSince.IsZero() && t.have && s.tx > t.prevTx && s.rx == t.prevRx:
			t.stallSince, t.stallRx = now, s.rx
		}
		stalled := !t.stallSince.IsZero() && now.Sub(t.stallSince) >= stallWindow
		judged := now.Sub(t.start) >= handshakeGrace
		var bad bool
		if s.handshake.IsZero() {
			bad = s.keepalive || s.tx > 0 || stalled
		} else {
			bad = s.keepalive || stalled
		}
		state = vpn.HealthIdle
		if bad && judged {
			state = vpn.HealthNoResponse
		}
	}
	t.have, t.prevRx, t.prevTx = true, s.rx, s.tx
	return t.set(now, vpn.TunnelHealth{State: state, LastHandshake: s.handshake})
}

// set stores h, keeping Since while the state does not change.
func (t *healthTracker) set(now time.Time, h vpn.TunnelHealth) vpn.TunnelHealth {
	if t.cur.State == h.State && !t.cur.Since.IsZero() {
		h.Since = t.cur.Since
	} else {
		h.Since = now
	}
	if h.State == vpn.HealthUnknown && h.LastHandshake.IsZero() {
		h.LastHandshake = t.cur.LastHandshake
	}
	t.cur = h
	return h
}

// totals returns the last counters read (rx, tx) and whether any were.
func (t *healthTracker) totals() (rx, tx uint64, ok bool) {
	return t.prevRx, t.prevTx, t.have
}
