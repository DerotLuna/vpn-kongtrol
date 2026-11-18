package daemonctl

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPanelURL_AppendsNonceFragment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/session/nonce" || r.Header.Get(TokenHeader) != "tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"nonce":"abc_-123","expires_in":60}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL, "tok").PanelURL("/app/tray")
	if err != nil {
		t.Fatal(err)
	}
	if want := srv.URL + "/app/tray#k=abc_-123"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPanelURL_FallsBackToPlainURL(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler()) // older daemon
	defer srv.Close()
	got, err := New(srv.URL, "tok").PanelURL("studio.html")
	if err == nil {
		t.Fatal("want error")
	}
	if got != srv.URL+"/studio.html" {
		t.Fatalf("got %q", got)
	}
}
