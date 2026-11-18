package daemonctl

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The CLI reads the DNS guard servers, the kill switch error and the
// blocked prefixes from the daemon (parity with the dashboards).
func TestSecurityAndBlocked(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/security/status":
			_, _ = w.Write([]byte(`{"dns_guard_servers":["10.0.0.53"],"kill_switch_error":{"message":"m","error_key":"killswitch.err.apply","detail":"boom"}}`))
		case "/api/v1/security/blocked":
			_, _ = w.Write([]byte(`{"prefixes":["10.20.0.0/16"],"count":1}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	c := New(ts.URL, "tok")
	st, err := c.Security()
	if err != nil || len(st.DNSGuardServers) != 1 || st.KillSwitchError == nil || st.KillSwitchError.ErrorKey != "killswitch.err.apply" {
		t.Fatalf("security = %+v, %v", st, err)
	}
	b, err := c.Blocked()
	if err != nil || len(b) != 1 || b[0] != "10.20.0.0/16" {
		t.Fatalf("blocked = %v, %v", b, err)
	}
}
