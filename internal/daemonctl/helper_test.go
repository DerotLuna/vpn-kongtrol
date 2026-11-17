package daemonctl

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestInstallHelper(t *testing.T) {
	var gotLang string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLang = r.Header.Get("X-Kongtrol-Lang")
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/helper/status" {
			_, _ = w.Write([]byte(`{"needs_install":true}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"No","hint":"Qué hacer: x","error_key":"service.elevation.cancelled"}`))
	}))
	defer ts.Close()
	c := New(ts.URL, "tok")

	st, err := c.HelperStatus()
	if err != nil || !st.NeedsInstall {
		t.Fatalf("status = %+v, %v", st, err)
	}
	_, err = c.InstallHelper(i18n.ES)
	var he *HelperInstallError
	if !errors.As(err, &he) || he.Key != "service.elevation.cancelled" || he.Hint != "Qué hacer: x" {
		t.Fatalf("err = %#v", err)
	}
	if gotLang != "es" {
		t.Fatalf("lang header = %q", gotLang)
	}
}
