package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

func buildTestPolicyEngine(t *testing.T) *policy.Engine {
	t.Helper()
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{"office": {Type: "openvpn"}},
		Policies: []config.PolicyRule{
			{Name: "office-net", Via: "office", Match: config.MatchSpec{Domains: []string{"example.com"}}},
		},
	}
	e, err := policy.New(cfg)
	if err != nil {
		t.Fatalf("policy.New: %v", err)
	}
	return e
}

// TestPolicyEngine_ConcurrentReloadAndResolve exercises the exact
// production concurrency pattern for s.policyEngine: one goroutine group
// hot-swaps the engine (what saveRuntimeConfig does on every policy CRUD
// request), while another concurrently serves resolve requests through the
// real HTTP handler — the same mix of traffic the dashboard can produce
// (someone editing a policy while another tab/request resolves a target).
// Run with `go test -race`: a plain unsynchronized *policy.Engine field
// would be flagged here; the atomic.Pointer[policy.Engine] field must not.
func TestPolicyEngine_ConcurrentReloadAndResolve(t *testing.T) {
	s := &Server{}
	s.policyEngine.Store(buildTestPolicyEngine(t))

	ts := httptest.NewServer(http.HandlerFunc(s.handleResolve))
	defer ts.Close()

	var wg sync.WaitGroup
	const iterations = 200

	// Writers: hot-swap the engine, mirroring saveRuntimeConfig's
	// s.policyEngine.Store(newEngine) under concurrent request load.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.policyEngine.Store(buildTestPolicyEngine(t))
			}
		}()
	}

	// Readers: real HTTP resolve requests, exercising handleResolve's
	// s.policyEngine.Load() + method calls on the loaded snapshot.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp, err := http.Get(ts.URL + "?target=example.com")
				if err != nil {
					t.Errorf("get: %v", err)
					return
				}
				_ = resp.Body.Close()
			}
		}()
	}

	wg.Wait()
}
