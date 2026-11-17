package api

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	brand "github.com/vpn-kongtrol/kongtrol/assets"
)

func TestBrandHandler(t *testing.T) {
	h := newBrandHandler(fstest.MapFS{
		"logo-kong.svg":        {Data: []byte("<svg/>")},
		"fonts/fonts.css":      {Data: []byte("@font-face{}")},
		"fonts/inter.woff2":    {Data: []byte("wOF2")},
		"generate.go":          {Data: []byte("package main")},
		"source/logo-kong.png": {Data: []byte("png")},
	})
	tests := []struct {
		method, path, wantType string
		wantStatus             int
	}{
		{http.MethodGet, "/brand/logo-kong.svg", "image/svg+xml", http.StatusOK},
		{http.MethodGet, "/brand/fonts/fonts.css", "text/css; charset=utf-8", http.StatusOK},
		{http.MethodGet, "/brand/fonts/inter.woff2", "font/woff2", http.StatusOK},
		{http.MethodHead, "/brand/logo-kong.svg", "image/svg+xml", http.StatusOK},
		{http.MethodGet, "/brand/generate.go", "", http.StatusNotFound},
		{http.MethodGet, "/brand/fonts/", "", http.StatusNotFound},
		{http.MethodGet, "/brand/", "", http.StatusNotFound},
		{http.MethodGet, "/brand/missing.png", "", http.StatusNotFound},
		{http.MethodPost, "/brand/logo-kong.svg", "", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != brandCacheControl {
				t.Errorf("Cache-Control = %q, want %q", got, brandCacheControl)
			}
		})
	}
}

// TestBrandEmbed_DashboardFiles guards the paths the legacy dashboard links
// (web/dashboard/*.html, shell.js) against renames in assets/brand.
func TestBrandEmbed_DashboardFiles(t *testing.T) {
	fsys := brand.Brand()
	for _, name := range []string{"logo-kong.svg", "logo.svg", "logo-kong-64.png", "fonts/fonts.css", "fonts/inter-latin-var.woff2"} {
		if _, err := fs.Stat(fsys, name); err != nil {
			t.Errorf("embedded brand file %q: %v", name, err)
		}
	}
	if _, err := fs.Stat(fsys, "generate.go"); err == nil {
		t.Error("generate.go must not be embedded")
	}
}
