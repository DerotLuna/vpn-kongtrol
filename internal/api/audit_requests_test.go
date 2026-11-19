package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every mutating route registered in Start has an audit action (or is a
// listed read-only POST), so a new endpoint cannot skip the audit log.
func TestAuditRoutes_CoverEveryMutatingRoute(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`mux\.HandleFunc\("((?:POST|PUT|PATCH|DELETE) [^"]+)"`)
	found := 0
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		found++
		if _, ok := auditRoutes[m[1]]; !ok && !auditSkip[m[1]] {
			t.Errorf("mutating route %q has no audit action (auditRoutes) and is not in auditSkip", m[1])
		}
	}
	for _, p := range []string{"POST " + sessionNoncePath, "POST " + sessionPath, "DELETE " + sessionPath} {
		if _, ok := auditRoutes[p]; !ok {
			t.Errorf("session route %q not audited", p)
		}
	}
	if found < 30 {
		t.Fatalf("only %d mutating routes found in server.go; did the registration change?", found)
	}
}

type auditSink struct {
	mu   sync.Mutex
	recs []AuditRecord
}

func (a *auditSink) add(r AuditRecord) {
	a.mu.Lock()
	a.recs = append(a.recs, r)
	a.mu.Unlock()
}

func (a *auditSink) all() []AuditRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]AuditRecord(nil), a.recs...)
}

func auditedServer(t *testing.T) (*Server, http.Handler, *auditSink) {
	t.Helper()
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "test-control-token"}
	sink := &auditSink{}
	s.SetAuditLog(sink.add)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/vpns/{name}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("name") == "bad" {
			http.Error(w, "nope", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("PUT /api/v1/policies/{name}", func(w http.ResponseWriter, _ *http.Request) {})
	mux.HandleFunc("POST /api/v1/policies/test", func(w http.ResponseWriter, _ *http.Request) {})
	mux.HandleFunc("GET /api/v1/tunnels", func(w http.ResponseWriter, _ *http.Request) {})
	return s, s.secureHandler(mux), sink
}

func send(h http.Handler, method, path, token, body string) int {
	req := httptest.NewRequest(method, "http://127.0.0.1:9741"+path, strings.NewReader(body))
	req.Host = "127.0.0.1:9741"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("X-Kongtrol-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// who / what / result, never the body (a profile update carries a password).
func TestAuditRequests_WhoWhatResultNoBody(t *testing.T) {
	s, h, sink := auditedServer(t)
	secret := `{"password":"hunter2-secret","private_key":"AAAA"}`
	send(h, http.MethodPut, "/api/v1/vpns/office", s.apiToken, secret)
	send(h, http.MethodPut, "/api/v1/vpns/bad", s.apiToken, secret)
	send(h, http.MethodPut, "/api/v1/policies/work", s.apiToken, `{}`)
	send(h, http.MethodPost, "/api/v1/policies/test", s.apiToken, `{}`) // read-only: skipped
	send(h, http.MethodGet, "/api/v1/tunnels", s.apiToken, "")          // read: skipped

	recs := sink.all()
	if len(recs) != 3 {
		t.Fatalf("got %d audit records, want 3: %+v", len(recs), recs)
	}
	ok, bad, pol := recs[0], recs[1], recs[2]
	if ok.Action != "config.profile.update" || ok.Profile != "office" || ok.Client != "app" || ok.Status != 200 || ok.Level != "SECURITY" {
		t.Errorf("profile update: %+v", ok)
	}
	if bad.Status != 400 || bad.Level != "WARN" || bad.Profile != "bad" {
		t.Errorf("refused update: %+v", bad)
	}
	if pol.Action != "config.policy.update" || pol.Name != "work" || pol.Profile != "" {
		t.Errorf("policy update: %+v", pol)
	}
	for _, r := range recs {
		if dump := fmt.Sprintf("%+v", r); strings.Contains(dump, "hunter2") || strings.Contains(dump, "AAAA") {
			t.Fatalf("a request body reached the audit record: %s", dump)
		}
	}
}

// Refused credentials are logged, at most one entry per window; the next
// entry says how many were skipped.
func TestAuditRequests_DeniedThrottled(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	srv, handler, sink := auditedServer(t)
	srv.authAudit.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if code := send(handler, http.MethodPut, "/api/v1/vpns/office", "wrong-token", `{}`); code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", code)
		}
	}
	now = now.Add(authAuditWindow)
	send(handler, http.MethodGet, "/api/v1/tunnels", "", "")
	recs := sink.all()
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (one per window): %+v", len(recs), recs)
	}
	if recs[0].Action != "auth.denied" || recs[0].Reason != "credential" || recs[0].Level != "SECURITY" || recs[0].Suppressed != 0 {
		t.Errorf("first: %+v", recs[0])
	}
	if recs[1].Suppressed != 4 || recs[1].Path != "/api/v1/tunnels" {
		t.Errorf("second should report 4 skipped: %+v", recs[1])
	}
}
