package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsTicketServer is a real loopback server with the session routes, the
// ticket route and a WebSocket route behind secureHandler (ADR-0062).
func wsTicketServer(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()
	s := &Server{bind: "127.0.0.1", apiToken: "file-token"}
	s.upgrader.CheckOrigin = s.checkWSOrigin
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+sessionNoncePath, s.handleSessionNonce)
	mux.HandleFunc("POST "+sessionPath, s.handleSessionCreate)
	mux.HandleFunc("POST "+wsTicketPath, s.handleWSTicket)
	mux.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
		c, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = c.WriteMessage(websocket.TextMessage, []byte("{}"))
		_ = c.Close()
	})
	ts := httptest.NewServer(s.secureHandler(mux))
	t.Cleanup(ts.Close)
	u, _ := url.Parse(ts.URL)
	s.port, _ = strconv.Atoi(u.Port())
	return s, ts, "http://" + u.Host
}

func post(t *testing.T, ts *httptest.Server, origin, path, body string, mod func(*http.Request)) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", origin)
	if mod != nil {
		mod(req)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// browserSession signs a page in the way web/ui does and returns its
// cookie and page token.
func browserSessionFor(t *testing.T, ts *httptest.Server, origin string) (*http.Cookie, string) {
	t.Helper()
	resp := post(t, ts, origin, sessionNoncePath, "", func(r *http.Request) {
		r.Header.Del("Origin")
		r.Header.Set("X-Kongtrol-Token", "file-token")
	})
	var n struct{ Nonce string }
	_ = json.NewDecoder(resp.Body).Decode(&n)
	resp = post(t, ts, origin, sessionPath, `{"nonce":"`+n.Nonce+`"}`, nil)
	var s struct {
		Token string `json:"session_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&s)
	if resp.StatusCode != http.StatusOK || s.Token == "" || len(resp.Cookies()) != 1 {
		t.Fatalf("exchange: %d %q", resp.StatusCode, s.Token)
	}
	return resp.Cookies()[0], s.Token
}

func ticketFor(t *testing.T, ts *httptest.Server, origin string, c *http.Cookie, tok string) (string, int) {
	t.Helper()
	resp := post(t, ts, origin, wsTicketPath, "", func(r *http.Request) {
		if c != nil {
			r.AddCookie(c)
		}
		if tok != "" {
			r.Header.Set(sessionHeader, tok)
		}
	})
	var out struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode == http.StatusOK && (out.Ticket == "" || out.ExpiresIn > 30) {
		t.Fatalf("ticket body: %+v", out)
	}
	return out.Ticket, resp.StatusCode
}

func dialWS(ts *httptest.Server, origin string, c *http.Cookie, ticket string) (int, error) {
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + wsPath
	if ticket != "" {
		u += "?" + wsTicketParam + "=" + url.QueryEscape(ticket)
	}
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	if c != nil {
		h.Set("Cookie", c.String())
	}
	conn, resp, err := websocket.DefaultDialer.Dial(u, h)
	if err != nil {
		if resp != nil {
			return resp.StatusCode, err
		}
		return 0, err
	}
	defer conn.Close()
	_, _, err = conn.ReadMessage()
	return http.StatusSwitchingProtocols, err
}

func TestWSTicketRequiresSessionHeader(t *testing.T) {
	_, ts, origin := wsTicketServer(t)
	c, tok := browserSessionFor(t, ts, origin)
	if _, code := ticketFor(t, ts, origin, c, ""); code != http.StatusUnauthorized {
		t.Fatalf("cookie alone: %d", code)
	}
	if _, code := ticketFor(t, ts, origin, c, tok); code != http.StatusOK {
		t.Fatalf("cookie + header: %d", code)
	}
}

func TestWebSocketNeedsOneTimeTicket(t *testing.T) {
	s, ts, origin := wsTicketServer(t)
	c, tok := browserSessionFor(t, ts, origin)

	if code, _ := dialWS(ts, origin, c, ""); code != http.StatusUnauthorized {
		t.Fatalf("no ticket: %d", code)
	}
	ticket, _ := ticketFor(t, ts, origin, c, tok)
	if code, err := dialWS(ts, origin, c, ticket); code != http.StatusSwitchingProtocols || err != nil {
		t.Fatalf("valid ticket: %d %v", code, err)
	}
	if code, _ := dialWS(ts, origin, c, ticket); code != http.StatusUnauthorized {
		t.Fatalf("reused ticket: %d", code)
	}

	// Wrong Origin: refused, and the ticket is not spent on it.
	ticket, _ = ticketFor(t, ts, origin, c, tok)
	if code, _ := dialWS(ts, "http://127.0.0.1:1", c, ticket); code != http.StatusForbidden {
		t.Fatalf("foreign origin: %d", code)
	}
	// Without the cookie the ticket alone does not open the socket.
	if code, _ := dialWS(ts, origin, nil, ticket); code != http.StatusUnauthorized {
		t.Fatalf("ticket without cookie: %d", code)
	}
	// A ticket of another session does not work with this cookie.
	other, otherTok := browserSessionFor(t, ts, origin)
	otherTicket, _ := ticketFor(t, ts, origin, other, otherTok)
	if code, _ := dialWS(ts, origin, c, otherTicket); code != http.StatusUnauthorized {
		t.Fatalf("another session's ticket: %d", code)
	}

	// Expired.
	now := time.Now()
	s.sessions.mu.Lock()
	s.sessions.now = func() time.Time { return now }
	s.sessions.mu.Unlock()
	ticket, _ = ticketFor(t, ts, origin, c, tok)
	s.sessions.mu.Lock()
	s.sessions.now = func() time.Time { return now.Add(ticketTTL) }
	s.sessions.mu.Unlock()
	if code, _ := dialWS(ts, origin, c, ticket); code != http.StatusUnauthorized {
		t.Fatalf("expired ticket: %d", code)
	}
}

// A planted cookie sent before the real one blocks neither the ticket nor
// the upgrade (ADR-0062).
func TestWebSocketWithPlantedCookie(t *testing.T) {
	_, ts, origin := wsTicketServer(t)
	c, tok := browserSessionFor(t, ts, origin)
	junk := &http.Cookie{Name: sessionCookieName, Value: "junk"}
	resp := post(t, ts, origin, wsTicketPath, "", func(r *http.Request) {
		r.AddCookie(junk)
		r.AddCookie(c)
		r.Header.Set(sessionHeader, tok)
	})
	var out struct{ Ticket string }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || out.Ticket == "" {
		t.Fatalf("ticket with planted cookie: %d", resp.StatusCode)
	}
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + wsPath + "?" + wsTicketParam + "=" + url.QueryEscape(out.Ticket)
	conn, _, err := websocket.DefaultDialer.Dial(u, http.Header{
		"Origin": {origin},
		"Cookie": {junk.String() + "; " + c.String()},
	})
	if err != nil {
		t.Fatalf("upgrade with planted cookie: %v", err)
	}
	conn.Close()
}

// Native clients (CLI live view) keep using the file token, no Origin.
func TestWebSocketFileTokenStillWorks(t *testing.T) {
	_, ts, _ := wsTicketServer(t)
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + wsPath
	conn, _, err := websocket.DefaultDialer.Dial(u, http.Header{"X-Kongtrol-Token": []string{"file-token"}})
	if err != nil {
		t.Fatalf("token dial: %v", err)
	}
	conn.Close()
	// No token and no Origin: not a browser of this daemon (403).
	if _, resp, err := websocket.DefaultDialer.Dial(u, nil); err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("no credentials must be refused: %v", err)
	}
}
