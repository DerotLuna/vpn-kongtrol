package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newTestChecker(t *testing.T, current string, h http.HandlerFunc) (*Checker, *atomic.Int32) {
	t.Helper()
	testAllowHTTP = true
	t.Cleanup(func() { testAllowHTTP = false })
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	c := New(current, func() bool { return true })
	c.APIBase = srv.URL
	c.StatePath = filepath.Join(t.TempDir(), "update-check.json")
	return c, &hits
}

func release(tag string, pre bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/"+DefaultRepo+"/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		p := "false"
		if pre {
			p = "true"
		}
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://github.com/` + DefaultRepo + `/releases/tag/` + tag + `","draft":false,"prerelease":` + p + `,"body":"notes"}`))
	}
}

func TestCheck_Newer(t *testing.T) {
	var ua, cookie, auth string
	c, _ := newTestChecker(t, "v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		ua, cookie, auth = r.UserAgent(), r.Header.Get("Cookie"), r.Header.Get("Authorization")
		release("v1.2.0", false)(w, r)
	})
	r, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !r.Newer || r.Latest != "v1.2.0" || r.URL == "" {
		t.Fatalf("want newer v1.2.0, got %+v", r)
	}
	if ua != "kongtrol/v1.0.0" || cookie != "" || auth != "" {
		t.Fatalf("request must carry only a generic User-Agent: ua=%q cookie=%q auth=%q", ua, cookie, auth)
	}
	if _, ok := c.TakeNotice(r); !ok {
		t.Fatal("first notice for v1.2.0 expected")
	}
	if _, ok := c.TakeNotice(r); ok {
		t.Fatal("v1.2.0 must be announced only once")
	}
}

func TestCheck_SameAndOlder(t *testing.T) {
	for _, tc := range []struct{ cur, latest string }{{"v1.2.0", "v1.2.0"}, {"v1.3.0", "v1.2.0"}, {"dev", "v1.2.0"}} {
		c, _ := newTestChecker(t, tc.cur, release(tc.latest, false))
		r, err := c.Check(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if r.Newer {
			t.Fatalf("%s vs %s: no notice expected", tc.cur, tc.latest)
		}
	}
}

func TestCheck_PrereleaseIgnored(t *testing.T) {
	for _, h := range []http.HandlerFunc{release("v2.0.0", true), release("v2.0.0-rc.1", false)} {
		c, _ := newTestChecker(t, "v1.0.0", h)
		r, _ := c.Check(context.Background())
		if r.Newer || r.Latest != "" {
			t.Fatalf("prerelease must be ignored, got %+v", r)
		}
	}
}

func TestCheck_Timeout(t *testing.T) {
	c, _ := newTestChecker(t, "v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	c.Client = &http.Client{Timeout: 100 * time.Millisecond}
	start := time.Now()
	if _, err := c.Check(context.Background()); err == nil {
		t.Fatal("timeout must be an error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("request not bounded by the timeout")
	}
}

func TestCheck_OncePerDay(t *testing.T) {
	c, hits := newTestChecker(t, "v1.0.0", release("v1.1.0", false))
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if _, err := c.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("want 1 request within 24h, got %d", hits.Load())
	}
	now = now.Add(Interval + time.Minute)
	r, _ := c.Check(context.Background())
	if hits.Load() != 2 || !r.Newer {
		t.Fatalf("want a second request after 24h, got %d", hits.Load())
	}
}

func TestCheck_FailureStillCountsTowardLimit(t *testing.T) {
	c, hits := newTestChecker(t, "v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	})
	_, _ = c.Check(context.Background())
	_, _ = c.Check(context.Background())
	if hits.Load() != 1 {
		t.Fatalf("a failed request must not be retried before 24h, got %d", hits.Load())
	}
}

func TestCheck_DisabledMakesNoRequest(t *testing.T) {
	c, hits := newTestChecker(t, "v1.0.0", release("v9.0.0", false))
	c.Enabled = func() bool { return false }
	if _, err := c.Check(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("want ErrDisabled, got %v", err)
	}
	c.Enabled = nil
	_, _ = c.Check(context.Background())
	if hits.Load() != 0 {
		t.Fatalf("disabled check made %d requests", hits.Load())
	}
}

func TestCheck_RejectsPlainHTTP(t *testing.T) {
	c := New("v1.0.0", func() bool { return true })
	c.APIBase = "http://api.github.com"
	c.StatePath = filepath.Join(t.TempDir(), "s.json")
	if _, err := c.Check(context.Background()); err == nil {
		t.Fatal("plain http must be refused")
	}
}

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, lat string
		want     bool
	}{
		{"v1.0.1-dev", "v1.0.1", true},
		{"1.0.0", "1.0.1", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "garbage", false},
		{"v1.0.0", "", false},
	}
	for _, tc := range cases {
		if got := Newer(tc.cur, tc.lat); got != tc.want {
			t.Errorf("Newer(%q,%q)=%v want %v", tc.cur, tc.lat, got, tc.want)
		}
	}
}

func TestLastFromNoRequest(t *testing.T) {
	r := LastFrom("v1.0.0", filepath.Join(t.TempDir(), "missing.json"))
	if r.Newer || r.Latest != "" {
		t.Fatalf("empty state expected, got %+v", r)
	}
}
