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
	tests := []struct {
		path, wantType, wantCache, wantBody string
		wantStatus                          int
	}{
		{"/app/", "text/html; charset=utf-8", "no-cache", "<title>ui</title>", http.StatusOK},
		{"/app/assets/index-abc.js", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable", "console.log", http.StatusOK},
		{"/app/assets/index-abc.css", "text/css; charset=utf-8", "public, max-age=31536000, immutable", "body{}", http.StatusOK},
		// Client-side route falls back to the SPA entry point.
		{"/app/settings/profiles", "text/html; charset=utf-8", "no-cache", "<title>ui</title>", http.StatusOK},
		// Missing hashed asset is a real 404, never HTML.
		{"/app/assets/missing.js", "", "", "", http.StatusNotFound},
		// Traversal attempts stay inside the UI FS.
		{"/app/../../go.mod", "text/html; charset=utf-8", "no-cache", "<title>ui</title>", http.StatusOK},
		// Directories are never listed.
		{"/app/assets/", "", "", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
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

func TestUIHandler_RejectsMutations(t *testing.T) {
	rec := httptest.NewRecorder()
	newUIHandler(uiTestFS(), true).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/app/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestUIHandler_NotBuiltIsTranslated(t *testing.T) {
	h := newUIHandler(fstest.MapFS{".gitkeep": {}}, false)
	req := httptest.NewRequest(http.MethodGet, "/app/", nil)
	req.Header.Set("Accept-Language", "es-AR,es;q=0.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="es"`, "make ui-build", "Qué hacer:", `href="/"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
}

// /app/ pages stay outside the API auth gate (like "/") but never hand
// out a credential: the session comes only from POST /api/v1/session.
func TestServer_UIRouteSetsNoCredentials(t *testing.T) {
	s := &Server{apiToken: "tok", port: 9741, bind: "127.0.0.1"}
	rec := httptest.NewRecorder()
	h := s.secureHandler(newUIHandler(uiTestFS(), true))
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9741/app/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cookies = %+v, want none", cookies)
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("secureHandler headers missing on /app/")
	}
}
