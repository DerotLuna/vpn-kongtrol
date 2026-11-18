package update

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

const good = "https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3"

func TestValidReleaseURL(t *testing.T) {
	ok := []string{good, "https://GitHub.com/DerotLuna/vpn-kongtrol/releases/tag/v0.10.0"}
	bad := []string{
		"",
		"http://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com@evil.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://user:pw@github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com.evil.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com./DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com:443/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://xn--gthub-n4a.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://gіthub.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3", // Cyrillic і
		"https://evil.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		good + "?x=1",
		good + "#frag",
		good + "/",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/../../../evil/repo/releases/tag/v1.2.3",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3/../../evil",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3%2F..%2Fx",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/%2e%2e",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3%5C",
		"https://github.com\\@evil.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com/Evil/vpn-kongtrol/releases/tag/v1.2.3",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3-rc.1",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v01.2.3",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/1.2.3",
		"https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3 ",
		"javascript:alert(1)",
		"//evil.com/DerotLuna/vpn-kongtrol/releases/tag/v1.2.3",
	}
	for _, u := range ok {
		if !ValidReleaseURL(u) {
			t.Errorf("rejected %q", u)
		}
	}
	for _, u := range bad {
		if ValidReleaseURL(u) {
			t.Errorf("accepted %q", u)
		}
	}
	if ReleaseURL("v1.2.3") != good || !ValidReleaseURL(ReleaseURL("v1.2.3")) {
		t.Errorf("ReleaseURL = %q", ReleaseURL("v1.2.3"))
	}
	if ReleaseURL("v1.2.3-rc.1") != "" || ReleaseURL("v1.2.3+meta") != "" {
		t.Error("ReleaseURL built a page for a non-strict tag")
	}
}

// Regression: html_url from the API answer was shown/opened if it was https.
func TestCheck_IgnoresHTMLURL(t *testing.T) {
	testAllowHTTP = true
	t.Cleanup(func() { testAllowHTTP = false })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v9.0.0","html_url":"https://evil.example/phish","draft":false,"prerelease":false}`))
	}))
	defer srv.Close()
	c := New("v1.0.0", func() bool { return true })
	c.APIBase = srv.URL
	c.Client = srv.Client()
	c.StatePath = filepath.Join(t.TempDir(), "s.json")
	r, err := c.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != ReleasePagePrefix+"v9.0.0" {
		t.Fatalf("URL = %q", r.URL)
	}
}

// A tampered state file is not trusted.
func TestLoadState_DropsTamperedValues(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"latest":"v9.9.9-evil","url":"https://evil.example/"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st := LoadState(p)
	if st.Latest != "" || st.URL != "" {
		t.Fatalf("state = %+v", st)
	}
	if err := os.WriteFile(p, []byte(`{"latest":"v2.0.0","url":"https://github.com@evil.com/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := LastFrom("v1.0.0", p); r.URL != ReleasePagePrefix+"v2.0.0" {
		t.Fatalf("URL = %q", r.URL)
	}
}

func TestCheckRedirect(t *testing.T) {
	mk := func(s string) *http.Request { u, _ := url.Parse(s); return &http.Request{URL: u} }
	if err := checkRedirect(mk("https://api.github.com/repositories/1/releases/latest"), nil); err != nil {
		t.Errorf("api.github.com refused: %v", err)
	}
	for _, s := range []string{"http://api.github.com/x", "https://evil.com/x", "https://api.github.com.evil.com/x", "https://api.github.com:8443/x", "https://u@api.github.com/x"} {
		if checkRedirect(mk(s), nil) == nil {
			t.Errorf("redirect to %s allowed", s)
		}
	}
	if checkRedirect(mk("https://api.github.com/x"), make([]*http.Request, 3)) == nil {
		t.Error("redirect loop allowed")
	}
}
