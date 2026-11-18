package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// Text assets go out gzip-compressed when the browser accepts it (Vary set
// either way); small files, binary types and clients without gzip get the
// plain bytes.
func TestUIHandler_Gzip(t *testing.T) {
	bigJS := strings.Repeat("export const a = 'kongtrol';\n", 200)
	fsys := fstest.MapFS{
		"index.html":          {Data: []byte("<!doctype html>" + strings.Repeat("<p>x</p>", 300))},
		"assets/app-1.js":     {Data: []byte(bigJS)},
		"assets/tiny-1.js":    {Data: []byte("x()")},
		"assets/font-1.woff2": {Data: []byte(strings.Repeat("A", 4096))},
	}
	h := newUIHandler(fsys, true)
	get := func(path, ae string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if ae != "" {
			req.Header.Set("Accept-Encoding", ae)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < 2; i++ { // second time from the cache
		rec := get("/assets/app-1.js", "br, gzip;q=0.8")
		if rec.Header().Get("Content-Encoding") != "gzip" || rec.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatalf("headers = %v", rec.Header())
		}
		if rec.Body.Len() >= len(bigJS) {
			t.Fatalf("not smaller: %d", rec.Body.Len())
		}
		zr, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		plain, _ := io.ReadAll(zr)
		if string(plain) != bigJS {
			t.Fatal("gzip body does not decode to the file")
		}
		if rec.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
			t.Fatalf("type = %q", rec.Header().Get("Content-Type"))
		}
	}
	if rec := get("/settings", "gzip"); rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("SPA index.html not compressed")
	}
	for _, c := range []struct{ path, ae string }{
		{"/assets/app-1.js", ""},
		{"/assets/app-1.js", "gzip;q=0, br"},
		{"/assets/app-1.js", "identity"},
		{"/assets/tiny-1.js", "gzip"},
		{"/assets/font-1.woff2", "gzip"},
	} {
		rec := get(c.path, c.ae)
		if rec.Header().Get("Content-Encoding") != "" || rec.Code != http.StatusOK {
			t.Fatalf("%s %q: encoded %q", c.path, c.ae, rec.Header().Get("Content-Encoding"))
		}
		if want := string(fsys[strings.TrimPrefix(c.path, "/")].Data); rec.Body.String() != want {
			t.Fatalf("%s: plain body differs", c.path)
		}
	}
	if v := get("/assets/font-1.woff2", "gzip").Header().Get("Vary"); v != "" {
		t.Fatalf("binary asset Vary = %q", v)
	}
	if !acceptsGzip("*") || acceptsGzip("*;q=0") || acceptsGzip("br") || !acceptsGzip("GZIP") {
		t.Fatal("acceptsGzip")
	}
}
