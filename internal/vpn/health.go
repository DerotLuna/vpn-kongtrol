package vpn

import "time"

// HealthState is whether a connected tunnel actually carries traffic, for
// adapters that can tell (WireGuard: the peer's latest handshake). It
// refines Status: an adapter whose tunnel stopped responding reports
// StatusError with HealthNoResponse, so the watchdog, kill switch and
// on_tunnel_down treat it like a drop (ADR-0030).
type HealthState string

const (
	// HealthOK: the server answered recently (handshake younger than
	// REJECT_AFTER_TIME + margin).
	HealthOK HealthState = "ok"
	// HealthIdle: no recent answer, but nothing was sent that needed one
	// (no traffic, no PersistentKeepalive). Normal for an unused tunnel.
	HealthIdle HealthState = "idle"
	// HealthNoResponse: Kongtrol's side kept sending (keepalive or
	// traffic) and the server did not answer.
	HealthNoResponse HealthState = "no_response"
	// HealthUnknown: the counters could not be read (e.g. the Kongtrol
	// helper is missing or outdated). The tunnel interface is up, but
	// Kongtrol cannot confirm the server answers.
	HealthUnknown HealthState = "unknown"
)

// TunnelHealth is the user-facing health of a connected tunnel. Clients
// turn it into a translated one-liner with MessageKey + i18n.
type TunnelHealth struct {
	State HealthState `json:"state"`
	// LastHandshake is the latest completed handshake (zero = never or
	// unknown).
	LastHandshake time.Time `json:"last_handshake,omitzero"`
	// Since is when the current State began.
	Since time.Time `json:"since,omitzero"`
	// ReasonKey is the i18n key of a "what to do" line for HealthUnknown
	// (e.g. "service.error.outdated"); empty when there is none.
	ReasonKey string `json:"reason_key,omitempty"`
}

// MessageKey returns the i18n key ("health.*", ES + EN in internal/i18n)
// of a one-line message about h, or "" when there is nothing to say.
// Clients append ReasonKey's text when set.
func (h TunnelHealth) MessageKey() string {
	switch h.State {
	case HealthIdle:
		return "health.idle"
	case HealthNoResponse:
		return "health.no_response"
	case HealthUnknown:
		return "health.unknown"
	}
	return ""
}

// Problem reports whether the user should look at h (shown highlighted).
func (h TunnelHealth) Problem() bool {
	return h.State == HealthNoResponse || h.State == HealthUnknown
}

// HealthReporter is implemented by adapters that can tell whether a
// connected tunnel actually responds. TunnelHealth returns nil when there
// is nothing to report (e.g. the tunnel is down).
type HealthReporter interface {
	TunnelHealth() *TunnelHealth
}
