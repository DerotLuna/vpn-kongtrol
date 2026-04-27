package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLeakTester_CheckNow_Clean(t *testing.T) {
	// Serve a fake IP check endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1.2.3.4"))
	}))
	defer srv.Close()

	lt := NewLeakTester(60*time.Second, nil)
	lt.ipCheckURL = srv.URL

	result := lt.CheckNow()
	if result.HasLeak {
		t.Errorf("expected no leak, got: %s", result.Reason)
	}
	if result.PublicIP != "1.2.3.4" {
		t.Errorf("PublicIP = %q, want 1.2.3.4", result.PublicIP)
	}
}

func TestLeakTester_CheckNow_LeakDetected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("9.9.9.9")) // unexpected IP
	}))
	defer srv.Close()

	lt := NewLeakTester(60*time.Second, map[string]string{
		"office": "1.2.3.4", // expected office IP
	})
	lt.ipCheckURL = srv.URL

	result := lt.CheckNow()
	if !result.HasLeak {
		t.Error("expected leak to be detected (wrong public IP)")
	}
}

func TestLeakTester_CheckNow_ServerDown(t *testing.T) {
	lt := NewLeakTester(60*time.Second, nil)
	lt.ipCheckURL = "http://127.0.0.1:0" // nothing listening

	result := lt.CheckNow()
	// A network error is not classified as a leak.
	if result.HasLeak {
		t.Error("network error should not be classified as a leak")
	}
}

func TestLeakTester_LastResult_NilBeforeFirstCheck(t *testing.T) {
	lt := NewLeakTester(60*time.Second, nil)
	if lt.LastResult() != nil {
		t.Error("LastResult should be nil before any check")
	}
}
