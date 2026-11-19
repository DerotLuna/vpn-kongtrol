package appattach

import "sync"

// Provider adapters register the matcher of their tunnel interface, so
// code that has no profile yet (GET /api/v1/vpns/detect) can tell whether
// the provider's app has its tunnel up now. Only interface listing (and
// `scutil --nc list` on macOS) is used: no admin, no GUI automation.
var (
	matchersMu sync.RWMutex
	matchers   = map[string]Matcher{}
)

// RegisterMatcher records the matcher of adapter type typ (from the
// adapter package's init).
func RegisterMatcher(typ string, m Matcher) {
	matchersMu.Lock()
	matchers[typ] = m
	matchersMu.Unlock()
}

// ActiveTunnel reports whether the tunnel of adapter type typ is up now.
// known is false when typ has no registered matcher (not a provider
// adapter): the caller then cannot tell.
func ActiveTunnel(typ string) (t Tunnel, up, known bool) {
	matchersMu.RLock()
	m, ok := matchers[typ]
	matchersMu.RUnlock()
	if !ok {
		return Tunnel{}, false, false
	}
	t, up = (&Session{}).Detect(m)
	return t, up, true
}
