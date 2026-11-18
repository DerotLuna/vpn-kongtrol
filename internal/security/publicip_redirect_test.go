package security

import (
	"net/http"
	"net/url"
	"testing"
)

// The public IP lookup may follow a redirect only over https and only to
// the endpoint's own host (never a downgrade to http or another service).
func TestPublicIPRedirect(t *testing.T) {
	mk := func(s string) *http.Request { u, _ := url.Parse(s); return &http.Request{URL: u} }
	via := []*http.Request{mk("https://api.ipify.org/")}
	if err := publicIPRedirect(mk("https://api.ipify.org/?format=text"), via); err != nil {
		t.Errorf("same-host https refused: %v", err)
	}
	for _, s := range []string{"http://api.ipify.org/", "https://evil.example/", "https://u@api.ipify.org/"} {
		if publicIPRedirect(mk(s), via) == nil {
			t.Errorf("redirect to %s allowed", s)
		}
	}
	if publicIPRedirect(mk("https://api.ipify.org/"), make([]*http.Request, 3)) == nil {
		t.Error("redirect loop allowed")
	}
}
