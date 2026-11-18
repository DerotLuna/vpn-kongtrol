package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ipServer is a TLS endpoint answering body; hits counts requests.
func ipServer(t *testing.T, body string, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTester(srv *httptest.Server, opts LeakTesterOptions) *LeakTester {
	opts.Endpoints = []string{srv.URL}
	opts.Client = srv.Client()
	return NewLeakTester(opts)
}

func TestLeakTester_CheckNow_Clean(t *testing.T) {
	lt := newTester(ipServer(t, "1.2.3.4\n", nil), LeakTesterOptions{Enabled: true})
	result := lt.CheckNow()
	if result.HasLeak || result.Disabled {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.PublicIP != "1.2.3.4" {
		t.Errorf("PublicIP = %q, want 1.2.3.4", result.PublicIP)
	}
	if lr := lt.LastResult(); lr == nil || lr.PublicIP != "1.2.3.4" {
		t.Errorf("LastResult = %+v", lr)
	}
}

func TestLeakTester_CheckNow_LeakDetected(t *testing.T) {
	lt := newTester(ipServer(t, "9.9.9.9", nil), LeakTesterOptions{
		Enabled:     true,
		ExpectedIPs: map[string]string{"office": "1.2.3.4"},
	})
	if result := lt.CheckNow(); !result.HasLeak {
		t.Error("expected leak to be detected (wrong public IP)")
	}
}

// Only connected profiles count: a profile that is off is never "the
// expected egress", and any connected profile's expected IP is a match.
func TestLeakVerdict_OnlyConnectedProfiles(t *testing.T) {
	expected := map[string]string{"office": "1.2.3.4", "home": "5.6.7.8"}
	on := func(names ...string) func() []string { return func() []string { return names } }
	cases := []struct {
		name   string
		ip     string
		active func() []string
		leak   bool
	}{
		{"nothing connected", "9.9.9.9", on(), false},
		{"connected without expected IP", "9.9.9.9", on("other"), false},
		{"office up, office IP", "1.2.3.4", on("office"), false},
		{"office up, other IP", "9.9.9.9", on("office"), true},
		{"home off is ignored", "1.2.3.4", on("office"), false},
		{"both up, home IP", "5.6.7.8", on("office", "home"), false},
		{"both up, neither", "9.9.9.9", on("office", "home"), true},
	}
	for _, c := range cases {
		leak, reason := leakVerdict(c.ip, expected, c.active)
		if leak != c.leak || (leak && reason == "") {
			t.Errorf("%s: leak=%v (%q), want %v", c.name, leak, reason, c.leak)
		}
	}
}

func TestLeakTester_CheckNow_ServerDown(t *testing.T) {
	lt := NewLeakTester(LeakTesterOptions{Enabled: true, Endpoints: []string{"https://127.0.0.1:1"}})
	result := lt.CheckNow()
	if result.HasLeak || result.PublicIP != "" || result.Reason == "" {
		t.Errorf("network error should be an error, not a leak: %+v", result)
	}
}

// Disabled (the default): no request at all, even when asked directly.
func TestLeakTester_Disabled_NoHTTP(t *testing.T) {
	var hits atomic.Int32
	lt := newTester(ipServer(t, "1.2.3.4", &hits), LeakTesterOptions{SettleDelay: -1})
	if r := lt.CheckNow(); !r.Disabled {
		t.Errorf("CheckNow on disabled tester = %+v, want Disabled", r)
	}
	lt.Trigger()
	time.Sleep(50 * time.Millisecond)
	if n := hits.Load(); n != 0 {
		t.Fatalf("disabled tester made %d HTTP requests", n)
	}
	if lt.LastResult() != nil {
		t.Error("LastResult should be nil while disabled")
	}
}

func TestLeakTester_RejectsNonIPAndLargeBodies(t *testing.T) {
	for _, body := range []string{"<html>hello</html>", strings.Repeat("1", 200), ""} {
		lt := newTester(ipServer(t, body, nil), LeakTesterOptions{Enabled: true})
		if r := lt.CheckNow(); r.PublicIP != "" || r.Reason == "" {
			t.Errorf("body %q: got %+v, want an error", body, r)
		}
	}
}

func TestLeakTester_RejectsHTTPEndpoint(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("1.2.3.4"))
	}))
	defer srv.Close()
	lt := NewLeakTester(LeakTesterOptions{Enabled: true, Endpoints: []string{srv.URL}, Client: srv.Client()})
	if r := lt.CheckNow(); r.PublicIP != "" {
		t.Errorf("http:// endpoint accepted: %+v", r)
	}
	if hits.Load() != 0 {
		t.Error("http:// endpoint was contacted")
	}
}

func TestLeakTester_FallsBackToNextEndpoint(t *testing.T) {
	good := ipServer(t, "2001:db8::1", nil)
	lt := NewLeakTester(LeakTesterOptions{
		Enabled:   true,
		Endpoints: []string{"https://127.0.0.1:1", good.URL},
		Client:    good.Client(),
	})
	if r := lt.CheckNow(); r.PublicIP != "2001:db8::1" {
		t.Errorf("fallback result = %+v", r)
	}
}

// Trigger coalesces bursts and keeps automatic lookups MinInterval apart.
func TestLeakTester_TriggerRateLimited(t *testing.T) {
	var hits atomic.Int32
	lt := newTester(ipServer(t, "1.2.3.4", &hits), LeakTesterOptions{
		Enabled:     true,
		MinInterval: 300 * time.Millisecond,
		SettleDelay: -1,
	})
	defer lt.Stop()
	for i := 0; i < 5; i++ {
		lt.Trigger()
	}
	waitHits(t, &hits, 1, time.Second)
	lt.Trigger() // inside the window: deferred, not immediate
	time.Sleep(100 * time.Millisecond)
	if n := hits.Load(); n != 1 {
		t.Fatalf("lookups inside MinInterval = %d, want 1", n)
	}
	waitHits(t, &hits, 2, time.Second) // runs once when the window ends
	time.Sleep(400 * time.Millisecond)
	if n := hits.Load(); n != 2 {
		t.Fatalf("lookups = %d, want 2", n)
	}
}

func TestLeakTester_ConfigureOffClearsResult(t *testing.T) {
	lt := newTester(ipServer(t, "1.2.3.4", nil), LeakTesterOptions{Enabled: true})
	lt.CheckNow()
	lt.Configure(false, nil, nil)
	if lt.LastResult() != nil {
		t.Error("disabling should forget the last public IP")
	}
}

func waitHits(t *testing.T, hits *atomic.Int32, want int32, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if hits.Load() >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("hits = %d, want %d", hits.Load(), want)
}
