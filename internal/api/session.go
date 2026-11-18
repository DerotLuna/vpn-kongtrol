package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Browser sessions (ADR-0039). The control token in ~/.kongtrol/api-token
// is for native clients (CLI, tray) only and never reaches a browser. A
// native client asks for a one-time nonce (POST /api/v1/session/nonce,
// token header), opens the page with it in the URL fragment (#k=…), and
// the page trades it for an HttpOnly session cookie (POST
// /api/v1/session). Sessions live in memory: a daemon restart ends them.
const (
	sessionCookieName = "kongtrol_session"
	sessionNoncePath  = "/api/v1/session/nonce"
	sessionPath       = "/api/v1/session"

	nonceTTL           = 60 * time.Second
	sessionIdleTimeout = 12 * time.Hour
	sessionMaxLifetime = 7 * 24 * time.Hour
	maxPendingNonces   = 32
	maxSessions        = 64

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
}

// sessionAuth stores only SHA-256 digests of nonces and session ids: the
// map lookup then compares digests, so its timing says nothing useful
// about the secret (same idea as comparing in constant time).
type sessionAuth struct {
	mu        sync.Mutex
	nonces    map[[32]byte]time.Time // digest -> expiry
	sessions  map[[32]byte]sessionEntry
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
	for k, exp := range a.nonces {
		if !now.Before(exp) {
			delete(a.nonces, k)
		}
	}
	if len(a.nonces) >= maxPendingNonces {
		var oldest [32]byte
		var oldestExp time.Time
		for k, exp := range a.nonces {
			if oldestExp.IsZero() || exp.Before(oldestExp) {
				oldest, oldestExp = k, exp
			}
		}
		delete(a.nonces, oldest)
	}
	a.nonces[digest(nonce)] = now.Add(nonceTTL)
	return nonce, nil
}

// exchangeResult is the outcome of exchange.
type exchangeResult int

const (
	exchangeOK exchangeResult = iota
	exchangeInvalid
	exchangeRateLimited
)

// exchange consumes nonce (once) and returns a new session id.
func (a *sessionAuth) exchange(nonce string) (string, exchangeResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if now.Sub(a.failStart) >= sessionFailWindow {
		a.failStart, a.failCount = now, 0
	}
	if a.failCount >= sessionFailLimit {
		return "", exchangeRateLimited
	}
	key := digest(nonce)
	exp, ok := a.nonces[key]
	if ok {
		delete(a.nonces, key) // single use, even when expired
	}
	if !ok || nonce == "" || !now.Before(exp) {
		a.failCount++
		return "", exchangeInvalid
	}
	id, err := newSecret()
	if err != nil {
		return "", exchangeInvalid
	}
	if a.sessions == nil {
		a.sessions = make(map[[32]byte]sessionEntry)
	}
	a.pruneSessionsLocked(now)
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
	a.sessions[digest(id)] = sessionEntry{created: now, lastSeen: now}
	return id, exchangeOK
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

// valid reports whether id is a live session and slides its idle timeout.
func (a *sessionAuth) valid(id string) bool {
	if id == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	key := digest(id)
	e, ok := a.sessions[key]
	if !ok {
		return false
	}
	now := a.clock()
	if a.expired(e, now) {
		delete(a.sessions, key)
		return false
	}
	e.lastSeen = now
	a.sessions[key] = e
	return true
}

func (a *sessionAuth) revoke(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, digest(id))
}

// handleSessionNonce answers POST /api/v1/session/nonce. secureHandler
// only lets it through with the file token in a header, never with a
// session cookie, so a browser page cannot mint more sessions.
func (s *Server) handleSessionNonce(w http.ResponseWriter, _ *http.Request) {
	nonce, err := s.sessions.issueNonce()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create nonce")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nonce":      nonce,
		"expires_in": int(nonceTTL / time.Second),
	})
}

// handleSessionCreate answers POST /api/v1/session {"nonce": "..."}: the
// only /api route reachable without credentials (Host, Origin and
// Content-Type are still checked by secureHandler).
func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nonce string `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	id, res := s.sessions.exchange(strings.TrimSpace(body.Nonce))
	switch res {
	case exchangeRateLimited:
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	case exchangeInvalid:
		writeError(w, http.StatusUnauthorized, "invalid or expired nonce")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleSessionDelete answers DELETE /api/v1/session: ends the browser
// session of the cookie sent (logout).
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
