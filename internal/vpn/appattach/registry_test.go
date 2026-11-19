package appattach

import "testing"

func TestActiveTunnel_Registry(t *testing.T) {
	if _, _, known := ActiveTunnel("no-such-type"); known {
		t.Fatal("a type without a matcher cannot be known")
	}
	RegisterMatcher("test-provider", Matcher{Patterns: []string{"kongtrol-test-none-*"}})
	t.Cleanup(func() {
		matchersMu.Lock()
		delete(matchers, "test-provider")
		matchersMu.Unlock()
	})
	if _, up, known := ActiveTunnel("test-provider"); !known || up {
		t.Fatalf("up=%v known=%v, want known and down", up, known)
	}
}
