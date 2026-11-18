package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func uiTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":           {Data: []byte("<!doctype html><title>ui</title>")},
		"assets/index-abc.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc.css": {Data: []byte("body{}")},
		".gitkeep":             {Data: nil},
	}
}

func TestUIHandler_Built(t *testing.T) {
	h := newUIHandler(uiTestFS(), true)
	const html, page = "text/html; charset=utf-8", "<title>ui</title>"
	tests := []struct {
		path, wantType, wantCache, wantBody string
		wantStatus                          int
	}{
		{"/", html, "no-cache", page, http.StatusOK},
		{"/index.html", html, "no-cache", page, http.StatusOK},
		{"/assets/index-abc.js", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable", "console.log", http.StatusOK},
		{"/assets/index-abc.css", "text/css; charset=utf-8", "public, max-age=31536000, immutable", "body{}", http.StatusOK},
		// Client-side routes (deep links, reloads) fall back to the SPA entry point.
		{"/settings", html, "no-cache", page, http.StatusOK},
		{"/profiles/groups", html, "no-cache", page, http.StatusOK},
		{"/rules/schedule?x=1", html, "no-cache", page, http.StatusOK},
		{"/tray", html, "no-cache", page, http.StatusOK},
		{"/apple", html, "no-cache", page, http.StatusOK}, // not the /app prefix
		// Missing files are a real 404, never HTML.
		{"/assets/missing.js", "", "", "", http.StatusNotFound},
		{"/favicon.ico", "", "", "", http.StatusNotFound},
		{"/old.html", "", "", "", http.StatusNotFound},
		{"/brand/logo-kong.svg", "", "", "", http.StatusNotFound},
		// An API path no route matched never gets the page.
		{"/api", "", "", "", http.StatusNotFound},
		{"/api/v1/nope", "", "", "", http.StatusNotFound},
		// Traversal attempts stay inside the UI FS.
		{"/../../go.mod", "", "", "", http.StatusNotFound},
		{"/../../web", html, "no-cache", page, http.StatusOK},
		// Directories are never listed.
		{"/assets/", "", "", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				if strings.Contains(rec.Body.String(), page) {
					t.Errorf("a %d answer must not be the page", rec.Code)
				}
				return
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// Old /app/ addresses and classic pages (bookmarks, older tray builds) move to the
// same page at "/". The "#k=" sign-in fragment never reaches the server;
// the browser keeps it across the redirect because Location has none.
func TestUIHandler_LegacyAppPrefixRedirects(t *testing.T) {
	for _, built := range []bool{true, false} {
		h := newUIHandler(uiTestFS(), built)
		tests := []struct{ path, want string }{
			{"/app", "/"},
			{"/app/", "/"},
			{"/app/tray", "/tray"},
			{"/app/rules/schedule", "/rules/schedule"},
			{"/app/settings?tab=2", "/settings?tab=2"},
			{"/app/assets/index-abc.js", "/assets/index-abc.js"},
			{"/app//evil.example", "/evil.example"},
			{"/app/%5Cevil.example", "/"},
			{"/app/../../x", "/x"},
			// Classic dashboard pages land on the section that replaced them.
			{"/studio.html", "/profiles"},
			{"/security.html", "/security"},
			{"/settings.html", "/settings"},
			{"/audit.html", "/activity"},
		}
		for _, tt := range tests {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusPermanentRedirect {
				t.Fatalf("built=%v %s: status = %d, want 308", built, tt.path, rec.Code)
			}
			loc := rec.Header().Get("Location")
			if loc != tt.want {
				t.Errorf("built=%v %s: Location = %q, want %q", built, tt.path, loc, tt.want)
			}
			if strings.Contains(loc, "#") || strings.HasPrefix(loc, "//") {
				t.Errorf("%s: Location %q must be a same-origin path without fragment", tt.path, loc)
			}
		}
	}
}

func TestUIHandler_RejectsMutations(t *testing.T) {
	for _, p := range []string{"/", "/settings", "/app/"} {
		rec := httptest.NewRecorder()
		newUIHandler(uiTestFS(), true).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, p, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s: status = %d, want 405", p, rec.Code)
		}
	}
}

func TestUIHandler_NotBuiltIsTranslated(t *testing.T) {
	h := newUIHandler(fstest.MapFS{".gitkeep": {}}, false)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("Accept-Language", "es-AR,es;q=0.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="es"`, "make ui-build", "Qué hacer:"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
	// Its one inline style is allowed by hash, never by 'unsafe-inline'.
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "style-src 'sha256-") || strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP = %q", csp)
	}
}

// Pages stay outside the API auth gate but never hand out a credential:
// the session comes only from POST /api/v1/session. The page CSP allows no
// inline script or style (the classic dashboard needed them; web/ui does not).
func TestServer_UIRouteSetsNoCredentials(t *testing.T) {
	s := &Server{apiToken: "tok", port: 9741, bind: "127.0.0.1"}
	h := s.secureHandler(newUIHandler(uiTestFS(), true))
	for _, p := range []string{"/", "/tray", "/settings"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9741"+p, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", p, rec.Code)
		}
		if cookies := rec.Result().Cookies(); len(cookies) != 0 {
			t.Fatalf("%s: cookies = %+v, want none", p, cookies)
		}
		csp := rec.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "script-src 'self';") || strings.Contains(csp, "unsafe-inline") {
			t.Errorf("%s: CSP = %q", p, csp)
		}
	}
	// Host validation is unchanged for pages.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://evil.example:9741/", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("foreign Host: status = %d, want 403", rec.Code)
	}
}
