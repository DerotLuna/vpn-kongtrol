package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Browser sessions (ADR-0039, bound to the page's origin by ADR-0062). The
// control token in ~/.kongtrol/api-token is for native clients (CLI, tray)
// only and never reaches a browser. A native client asks for a one-time
// nonce (POST /api/v1/session/nonce, token header), opens the page with it
// in the URL fragment (#k=…), and the page trades it (POST /api/v1/session)
// for an HttpOnly session cookie plus a per-page session token in the JSON
// body. Every request needs both: the cookie alone is not enough, because
// browsers send cookies to every port of 127.0.0.1 (RFC 6265 §8.5), so
// another local user's server on another port could capture it. The token
// lives in the page's sessionStorage, which only this origin (scheme, host
// and port) can read, and travels in X-Kongtrol-Session. The WebSocket (no
// custom headers in browsers) uses a one-time ticket from POST
// /api/v1/ws-ticket. Sessions live in memory: a daemon restart ends them.
const (
	sessionCookieName = "kongtrol_session"
	sessionHeader     = "X-Kongtrol-Session"
	sessionNoncePath  = "/api/v1/session/nonce"
	sessionPath       = "/api/v1/session"
	wsTicketPath      = "/api/v1/ws-ticket"
	wsPath            = "/api/v1/ws/metrics"
	wsTicketParam     = "ticket"

	nonceTTL           = 60 * time.Second
	ticketTTL          = 30 * time.Second
	sessionIdleTimeout = 12 * time.Hour
	sessionMaxLifetime = 7 * 24 * time.Hour
	maxPendingNonces   = 32
	maxPendingTickets  = 64
	maxSessions        = 64
	// maxPageTokens: pages (tabs, the tray flyout) sharing one cookie, each
	// with its own token; the oldest page signs out past it.
	maxPageTokens = 16

	// sessionFailLimit failed exchanges per sessionFailWindow (all
	// callers together) lock the exchange until the window ends. Nonces
	// have 256 bits, so this only stops noise, not a feasible attack.
	sessionFailLimit  = 10
	sessionFailWindow = time.Minute

	secretBytes = 32
)

type sessionEntry struct {
	created  time.Time
	lastSeen time.Time
	tokens   [][32]byte // digests of the page tokens of this cookie
}

type ticketEntry struct {
	session [32]byte
	expires time.Time
}

// sessionAuth stores only SHA-256 digests of nonces, session ids, page
// tokens and tickets: map lookups then compare digests, so their timing
// says nothing useful about the secret; page tokens are compared in
// constant time too.
type sessionAuth struct {
	mu        sync.Mutex
	nonces    map[[32]byte]time.Time // digest -> expiry
	sessions  map[[32]byte]sessionEntry
	tickets   map[[32]byte]ticketEntry
	failCount int
	failStart time.Time
	now       func() time.Time // test seam; nil = time.Now
}

func (a *sessionAuth) clock() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}

func newSecret() (string, error) {
	raw := make([]byte, secretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func digest(secret string) [32]byte { return sha256.Sum256([]byte(secret)) }

// issueNonce returns a new single-use nonce valid for nonceTTL.
func (a *sessionAuth) issueNonce() (string, error) {
	nonce, err := newSecret()
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if a.nonces == nil {
		a.nonces = make(map[[32]byte]time.Time)
	}
	pruneExpiring(a.nonces, now, func(exp time.Time) time.Time { return exp }, maxPendingNonces)
	a.nonces[digest(nonce)] = now.Add(nonceTTL)
	return nonce, nil
}

// pruneExpiring drops expired entries and, when the map is full, the one
// that expires first.
func pruneExpiring[V any](m map[[32]byte]V, now time.Time, exp func(V) time.Time, limit int) {
	for k, v := range m {
		if !now.Before(exp(v)) {
			delete(m, k)
		}
	}
	if len(m) < limit {
		return
	}
	var oldest [32]byte
	var oldestExp time.Time
	for k, v := range m {
		if e := exp(v); oldestExp.IsZero() || e.Before(oldestExp) {
			oldest, oldestExp = k, e
		}
	}
	delete(m, oldest)
}

// exchangeResult is the outcome of exchange.
type exchangeResult int

const (
	exchangeOK exchangeResult = iota
	exchangeInvalid
	exchangeRateLimited
)

// exchange consumes nonce (once) and returns the session id for the cookie
// and a new page token. A browser that already holds a live session cookie
// keeps it and gets one more page token, so a second tab (or the tray
// flyout) does not sign out the first one.
func (a *sessionAuth) exchange(nonce, cookie string) (id, token string, res exchangeResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if now.Sub(a.failStart) >= sessionFailWindow {
		a.failStart, a.failCount = now, 0
	}
	if a.failCount >= sessionFailLimit {
		return "", "", exchangeRateLimited
	}
	key := digest(nonce)
	exp, ok := a.nonces[key]
	if ok {
		delete(a.nonces, key) // single use, even when expired
	}
	if !ok || nonce == "" || !now.Before(exp) {
		a.failCount++
		return "", "", exchangeInvalid
	}
	token, err := newSecret()
	if err != nil {
		return "", "", exchangeInvalid
	}
	if a.sessions == nil {
		a.sessions = make(map[[32]byte]sessionEntry)
	}
	a.pruneSessionsLocked(now)
	if cookie != "" {
		if e, ok := a.sessions[digest(cookie)]; ok {
			e.tokens = append(e.tokens, digest(token))
			if len(e.tokens) > maxPageTokens {
				e.tokens = e.tokens[len(e.tokens)-maxPageTokens:]
			}
			e.lastSeen = now
			a.sessions[digest(cookie)] = e
			return cookie, token, exchangeOK
		}
	}
	id, err = newSecret()
	if err != nil {
		return "", "", exchangeInvalid
	}
	if len(a.sessions) >= maxSessions {
		var oldest [32]byte
		var oldestSeen time.Time
		for k, e := range a.sessions {
			if oldestSeen.IsZero() || e.lastSeen.Before(oldestSeen) {
				oldest, oldestSeen = k, e.lastSeen
			}
		}
		delete(a.sessions, oldest)
	}
	a.sessions[digest(id)] = sessionEntry{created: now, lastSeen: now, tokens: [][32]byte{digest(token)}}
	return id, token, exchangeOK
}

func (a *sessionAuth) pruneSessionsLocked(now time.Time) {
	for k, e := range a.sessions {
		if a.expired(e, now) {
			delete(a.sessions, k)
		}
	}
}

func (a *sessionAuth) expired(e sessionEntry, now time.Time) bool {
	return now.Sub(e.lastSeen) >= sessionIdleTimeout || now.Sub(e.created) >= sessionMaxLifetime
}

// liveLocked returns the live session of cookie id (expired ones are
// dropped).
func (a *sessionAuth) liveLocked(id string, now time.Time) (sessionEntry, bool) {
	if id == "" {
		return sessionEntry{}, false
	}
	key := digest(id)
	e, ok := a.sessions[key]
	if !ok {
		return sessionEntry{}, false
	}
	if a.expired(e, now) {
		delete(a.sessions, key)
		return sessionEntry{}, false
	}
	return e, true
}

// valid reports whether cookie id is a live session and token one of its
// page tokens, and slides its idle timeout.
func (a *sessionAuth) valid(id, token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	e, ok := a.liveLocked(id, now)
	if !ok {
		return false
	}
	want := digest(token)
	match := 0
	for _, t := range e.tokens {
		match |= subtle.ConstantTimeCompare(t[:], want[:])
	}
	if match != 1 {
		return false
	}
	e.lastSeen = now
	a.sessions[digest(id)] = e
	return true
}

func (a *sessionAuth) revoke(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, digest(id))
}

// issueTicket returns a one-time WebSocket ticket (ticketTTL) bound to the
// live session of cookie id.
func (a *sessionAuth) issueTicket(id string) (string, bool) {
	ticket, err := newSecret()
	if err != nil {
		return "", false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if _, ok := a.liveLocked(id, now); !ok {
		return "", false
	}
	if a.tickets == nil {
		a.tickets = make(map[[32]byte]ticketEntry)
	}
	pruneExpiring(a.tickets, now, func(t ticketEntry) time.Time { return t.expires }, maxPendingTickets)
	a.tickets[digest(ticket)] = ticketEntry{session: digest(id), expires: now.Add(ticketTTL)}
	return ticket, true
}

// consumeTicket uses ticket up (valid or not) and reports whether it was
// live and issued to the session of cookie id, which must still be live.
func (a *sessionAuth) consumeTicket(ticket, id string) bool {
	if ticket == "" || id == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	key := digest(ticket)
	t, ok := a.tickets[key]
	if !ok {
		return false
	}
	delete(a.tickets, key)
	sess := digest(id)
	if !now.Before(t.expires) || subtle.ConstantTimeCompare(t.session[:], sess[:]) != 1 {
		return false
	}
	_, live := a.liveLocked(id, now)
	return live
}

// handleSessionNonce answers POST /api/v1/session/nonce. secureHandler
// only lets it through with the file token in a header, never with a
// session, so a browser page cannot mint more sessions.
func (s *Server) handleSessionNonce(w http.ResponseWriter, _ *http.Request) {
	nonce, err := s.sessions.issueNonce()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create nonce")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"nonce":      nonce,
		"expires_in": int(nonceTTL / time.Second),
	})
}

// handleSessionCreate answers POST /api/v1/session {"nonce": "..."}: the
// only /api route reachable without credentials (Host, Origin and
// Content-Type are still checked by secureHandler). Answers
// {"session_token": "..."} and sets the cookie.
func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	cookie := ""
	if c, err := r.Cookie(sessionCookieName); err == nil {
		cookie = c.Value
	}
	id, token, res := s.sessions.exchange(strings.TrimSpace(body.Nonce), cookie)
	switch res {
	case exchangeRateLimited:
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	case exchangeInvalid:
		writeError(w, http.StatusUnauthorized, "invalid or expired nonce")
		return
	}
	// No Secure attribute: the page is plain http on loopback and Safari
	// still drops Secure cookies there (WebKit bug 281149, ADR-0062).
	// Without the page token in sessionStorage the cookie is worthless.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"session_token": token})
}

// handleSessionDelete answers DELETE /api/v1/session: ends the browser
// session of the cookie sent (logout, every page of that browser).
func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.revoke(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleWSTicket answers POST /api/v1/ws-ticket for a browser session
// (cookie + X-Kongtrol-Session, checked by secureHandler): a one-time
// ticket for GET /api/v1/ws/metrics?ticket=… within ticketTTL. Native
// clients send the file token on the WebSocket itself.
func (s *Server) handleWSTicket(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "browser sessions only")
		return
	}
	ticket, ok := s.sessions.issueTicket(c.Value)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_in": int(ticketTTL / time.Second),
	})
}

// browserSession reports whether r carries a live session cookie plus one
// of its page tokens in X-Kongtrol-Session.
func (s *Server) browserSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return s.sessions.valid(c.Value, strings.TrimSpace(r.Header.Get(sessionHeader)))
}

// wsTicketAuthenticated checks a browser WebSocket upgrade: the Origin of
// this daemon, the session cookie and a one-time ticket issued to it.
func (s *Server) wsTicketAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return s.sessions.consumeTicket(r.URL.Query().Get(wsTicketParam), c.Value)
}
