package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testOrigin = "http://127.0.0.1:9741"

func sessionTestServer() (*Server, http.Handler) {
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "file-token"}
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+sessionNoncePath, s.handleSessionNonce)
	mux.HandleFunc("POST "+sessionPath, s.handleSessionCreate)
	mux.HandleFunc("DELETE "+sessionPath, s.handleSessionDelete)
	mux.HandleFunc("POST /api/v1/data/purge", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("page"))
	}))
	return s, s.secureHandler(mux)
}

func doReq(h http.Handler, method, path, body string, mod func(*http.Request)) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, testOrigin+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, testOrigin+path, nil)
	}
	req.Host = "127.0.0.1:9741"
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func mintNonce(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := doReq(h, http.MethodPost, sessionNoncePath, "", func(r *http.Request) {
		r.Header.Set("X-Kongtrol-Token", "file-token")
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("nonce status=%d", rec.Code)
	}
	var out struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Nonce == "" {
		t.Fatalf("nonce body=%s", rec.Body.String())
	}
	return out.Nonce
}

func exchangeNonce(h http.Handler, nonce string) *httptest.ResponseRecorder {
	return doReq(h, http.MethodPost, sessionPath, `{"nonce":"`+nonce+`"}`, func(r *http.Request) {
		r.Header.Set("Origin", testOrigin)
	})
}

// Regression: GET / handed the file token to anyone as a cookie.
func TestPagesNeverSetCredentials(t *testing.T) {
	_, h := sessionTestServer()
	for _, p := range []string{"/", "/settings", "/tray"} {
		rec := doReq(h, http.MethodGet, p, "", nil)
		if len(rec.Result().Cookies()) != 0 {
			t.Fatalf("%s set cookies %v", p, rec.Result().Cookies())
		}
	}
}

func TestPagesValidateHost(t *testing.T) {
	_, h := sessionTestServer()
	rec := doReq(h, http.MethodGet, "/", "", func(r *http.Request) { r.Host = "evil.example:9741" })
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestNonceRequiresFileTokenHeader(t *testing.T) {
	s, h := sessionTestServer()
	if rec := doReq(h, http.MethodPost, sessionNoncePath, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status=%d", rec.Code)
	}
	// A browser session must not mint nonces.
	id, tok, _ := s.sessions.exchange(mintNonce(t, h), "")
	rec := doReq(h, http.MethodPost, sessionNoncePath, "", func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
		r.Header.Set(sessionHeader, tok)
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("session cookie: status=%d", rec.Code)
	}
}

func TestSessionExchangeFlow(t *testing.T) {
	_, h := sessionTestServer()
	nonce := mintNonce(t, h)
	rec := exchangeNonce(h, nonce)
	c, tok := sessionFrom(t, rec)
	if c.Name != sessionCookieName || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Fatalf("cookie attrs=%+v", c)
	}
	if c.Value == "file-token" || c.Value == nonce || tok == c.Value || tok == nonce {
		t.Fatal("session id and page token must differ from each other, the file token and the nonce")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("the page token must not be cached")
	}
	// Session works for the API.
	if code := purgeWith(h, c, tok); code != http.StatusNoContent {
		t.Fatalf("api with session: status=%d", code)
	}
	// Single use.
	if rec := exchangeNonce(h, nonce); rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused nonce: status=%d", rec.Code)
	}
	// Logout ends it.
	doReq(h, http.MethodDelete, sessionPath, "", func(r *http.Request) {
		r.AddCookie(c)
		r.Header.Set(sessionHeader, tok)
	})
	if code := purgeWith(h, c, tok); code != http.StatusUnauthorized {
		t.Fatalf("after logout: status=%d", code)
	}
}

// sessionFrom reads the cookie and page token of a successful exchange.
func sessionFrom(t *testing.T, rec *httptest.ResponseRecorder) (*http.Cookie, string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("exchange status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v", cookies)
	}
	var out struct {
		Token string `json:"session_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Token == "" {
		t.Fatalf("body=%s", rec.Body.String())
	}
	return cookies[0], out.Token
}

// purgeWith calls an API route with cookie c (nil = none) and page token
// tok ("" = no header).
func purgeWith(h http.Handler, c *http.Cookie, tok string) int {
	return doReq(h, http.MethodPost, "/api/v1/data/purge", `{"confirm":true}`, func(r *http.Request) {
		if c != nil {
			r.AddCookie(c)
		}
		if tok != "" {
			r.Header.Set(sessionHeader, tok)
		}
		r.Header.Set("Origin", testOrigin)
	}).Code
}

// Regression (ADR-0062): browsers send the cookie to every port of
// 127.0.0.1, so a server of another local user could capture it. The cookie
// alone, or with a wrong page token, must not authenticate.
func TestSessionCookieNeedsPageToken(t *testing.T) {
	_, h := sessionTestServer()
	c, tok := sessionFrom(t, exchangeNonce(h, mintNonce(t, h)))
	other, otherTok := sessionFrom(t, exchangeNonce(h, mintNonce(t, h)))
	cases := []struct {
		name string
		c    *http.Cookie
		tok  string
		want int
	}{
		{"cookie without header", c, "", http.StatusUnauthorized},
		{"cookie + wrong header", c, tok + "x", http.StatusUnauthorized},
		{"cookie + another session's token", c, otherTok, http.StatusUnauthorized},
		{"header without cookie", nil, tok, http.StatusUnauthorized},
		{"cookie + right header", c, tok, http.StatusNoContent},
		{"other cookie + its header", other, otherTok, http.StatusNoContent},
	}
	for _, tc := range cases {
		if got := purgeWith(h, tc.c, tc.tok); got != tc.want {
			t.Errorf("%s: status=%d, want %d", tc.name, got, tc.want)
		}
	}
}

// A second page (another tab, the tray flyout) of the same browser keeps
// the cookie and gets its own token: the first page stays signed in.
func TestSecondPageKeepsFirstSignedIn(t *testing.T) {
	_, h := sessionTestServer()
	c, first := sessionFrom(t, exchangeNonce(h, mintNonce(t, h)))
	rec := doReq(h, http.MethodPost, sessionPath, `{"nonce":"`+mintNonce(t, h)+`"}`, func(r *http.Request) {
		r.Header.Set("Origin", testOrigin)
		r.AddCookie(c)
	})
	c2, second := sessionFrom(t, rec)
	if c2.Value != c.Value || second == first {
		t.Fatalf("cookie must stay, token must be new: %q/%q %q/%q", c.Value, c2.Value, first, second)
	}
	for _, tok := range []string{first, second} {
		if got := purgeWith(h, c, tok); got != http.StatusNoContent {
			t.Fatalf("token %q: status=%d", tok, got)
		}
	}
	// A captured cookie cannot add pages without a nonce.
	rec = doReq(h, http.MethodPost, sessionPath, `{"nonce":"guess"}`, func(r *http.Request) {
		r.Header.Set("Origin", testOrigin)
		r.AddCookie(c)
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("exchange without a nonce: %d", rec.Code)
	}
}

func TestPageTokensPerSessionAreBounded(t *testing.T) {
	a := &sessionAuth{}
	n, _ := a.issueNonce()
	id, first, _ := a.exchange(n, "")
	for range maxPageTokens {
		n, _ = a.issueNonce()
		if got, _, res := a.exchange(n, id); res != exchangeOK || got != id {
			t.Fatalf("exchange with cookie: %v %q", res, got)
		}
	}
	if a.valid(id, first) {
		t.Fatal("the oldest page token must drop past maxPageTokens")
	}
}

func TestCSPNamesOnlyThisWebSocketOrigin(t *testing.T) {
	_, h := sessionTestServer()
	csp := doReq(h, http.MethodGet, "/", "", nil).Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self' ws://127.0.0.1:9741;") {
		t.Fatalf("csp=%q", csp)
	}
	if strings.Contains(csp, "ws: ") || strings.Contains(csp, "wss:") {
		t.Fatalf("any-host WebSocket sources must be gone: %q", csp)
	}
	csp = doReq(h, http.MethodGet, "/", "", func(r *http.Request) { r.Host = "evil.example:9741" }).Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "evil") {
		t.Fatalf("a foreign Host must not reach the CSP: %q", csp)
	}
}

func TestFileTokenNotAcceptedAsCookie(t *testing.T) {
	_, h := sessionTestServer()
	for _, name := range []string{"kongtrol_control", sessionCookieName} {
		rec := doReq(h, http.MethodPost, "/api/v1/data/purge", `{"confirm":true}`, func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: name, Value: "file-token"})
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s cookie with file token: status=%d", name, rec.Code)
		}
	}
}

func TestSessionExchangeRejectsHostileOrigin(t *testing.T) {
	_, h := sessionTestServer()
	nonce := mintNonce(t, h)
	rec := doReq(h, http.MethodPost, sessionPath, `{"nonce":"`+nonce+`"}`, func(r *http.Request) {
		r.Header.Set("Origin", "https://evil.example")
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
	rec = doReq(h, http.MethodPost, sessionPath, `{"nonce":"`+nonce+`"}`, func(r *http.Request) {
		r.Host = "evil.example:9741"
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad host status=%d", rec.Code)
	}
	for _, origin := range []string{"", "null"} {
		rec = doReq(h, http.MethodPost, sessionPath, `{"nonce":"`+nonce+`"}`, func(r *http.Request) {
			if origin != "" {
				r.Header.Set("Origin", origin)
			}
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("origin %q status=%d", origin, rec.Code)
		}
	}
}

func TestNonceExpiresAndSessionTimesOut(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := &sessionAuth{now: func() time.Time { return now }}
	n, _ := a.issueNonce()
	now = now.Add(nonceTTL)
	if _, _, res := a.exchange(n, ""); res != exchangeInvalid {
		t.Fatalf("expired nonce accepted")
	}
	n, _ = a.issueNonce()
	id, tok, res := a.exchange(n, "")
	if res != exchangeOK {
		t.Fatal("fresh nonce rejected")
	}
	now = now.Add(sessionIdleTimeout - time.Minute)
	if !a.valid(id, tok) { // slides
		t.Fatal("session expired early")
	}
	now = now.Add(sessionIdleTimeout - time.Minute)
	if !a.valid(id, tok) {
		t.Fatal("sliding expiry not applied")
	}
	now = now.Add(sessionIdleTimeout)
	if a.valid(id, tok) {
		t.Fatal("idle session still valid")
	}
}

func TestSessionAbsoluteLifetime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := &sessionAuth{now: func() time.Time { return now }}
	n, _ := a.issueNonce()
	id, tok, _ := a.exchange(n, "")
	for now.Sub(time.Unix(1_700_000_000, 0)) < sessionMaxLifetime-time.Hour {
		now = now.Add(time.Hour)
		if !a.valid(id, tok) {
			t.Fatal("expired before max lifetime")
		}
	}
	now = now.Add(2 * time.Hour)
	if a.valid(id, tok) {
		t.Fatal("valid after max lifetime")
	}
}

func TestSessionExchangeRateLimited(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := &sessionAuth{now: func() time.Time { return now }}
	for i := 0; i < sessionFailLimit; i++ {
		if _, _, res := a.exchange("wrong", ""); res != exchangeInvalid {
			t.Fatalf("attempt %d: %v", i, res)
		}
	}
	good, _ := a.issueNonce()
	if _, _, res := a.exchange(good, ""); res != exchangeRateLimited {
		t.Fatalf("want rate limited, got %v", res)
	}
	now = now.Add(sessionFailWindow - time.Second)
	if _, _, res := a.exchange(good, ""); res != exchangeRateLimited {
		t.Fatalf("still in window: %v", res)
	}
	now = now.Add(time.Second)
	good, _ = a.issueNonce()
	if _, _, res := a.exchange(good, ""); res != exchangeOK {
		t.Fatalf("after window: %v", res)
	}
}

func TestListenHostPinsLocalhost(t *testing.T) {
	for in, want := range map[string]string{"localhost": "127.0.0.1", "LOCALHOST.": "127.0.0.1", "127.0.0.1": "127.0.0.1", "::1": "::1"} {
		if got := listenHost(in); got != want {
			t.Errorf("listenHost(%q)=%q, want %q", in, got, want)
		}
	}
}
