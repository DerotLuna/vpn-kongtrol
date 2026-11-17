package security

import "strings"

// Kill switch states reported to every client (CLI, TUI, doctor, tray,
// dashboard), which render them translated:
//
//	off    INACTIVO — turned off (globally and for every profile)
//	armed  ARMADO   — protecting Protected (connected protected tunnels)
//	active ACTIVO   — blocking the destinations of Blocking (dropped tunnels)
const (
	KillSwitchOff    = "off"
	KillSwitchArmed  = "armed"
	KillSwitchActive = "active"
)

// KillSwitchStatus is the daemon's kill switch state.
type KillSwitchStatus struct {
	State string `json:"state"`
	// Protected are the connected tunnels the kill switch guards.
	Protected []string `json:"protected,omitempty"`
	// Blocking are the tunnels that dropped unexpectedly and whose
	// destinations are blocked now.
	Blocking []string `json:"blocking,omitempty"`
	// FullTunnel lists the Blocking tunnels that carried all traffic:
	// everything but their endpoints (and the LAN when allowed) is blocked.
	FullTunnel []string `json:"full_tunnel,omitempty"`
	// PolicyBlocking are policies with on_tunnel_down: block whose tunnel
	// is off, so their destinations are blocked.
	PolicyBlocking []string `json:"policy_blocking,omitempty"`
	// BlockedCount is the number of blocked prefixes in place.
	BlockedCount int `json:"blocked_count"`
	// Error is the last failure applying the block (English, for logs);
	// ErrorKey its translated "what to do" line when known.
	Error    string `json:"error,omitempty"`
	ErrorKey string `json:"error_key,omitempty"`
}

// StateKey is the i18n key of the one-line label ("killswitch.state.*",
// ES + EN in internal/i18n) and its arguments.
func (s KillSwitchStatus) StateKey() (string, []any) {
	switch s.State {
	case KillSwitchActive:
		return "killswitch.state.active", []any{strings.Join(s.Blocking, ", ")}
	case KillSwitchArmed:
		if len(s.Protected) == 0 {
			return "killswitch.state.armed_none", nil
		}
		return "killswitch.state.armed", []any{len(s.Protected)}
	}
	return "killswitch.state.off", nil
}
