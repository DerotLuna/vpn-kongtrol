package api

import (
	"bytes"
	"compress/gzip"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// uiPrefix is where the new UI (web/ui, embedded by web/ui_embed.go) is
// served. The legacy dashboard (web/dashboard) keeps "/" until every page
// has migrated; then the prefix can move to "/" in one place.
const uiPrefix = "/app/"

// uiContentTypes pins the types the UI build emits. mime.TypeByExtension
// consults the Windows registry, which can map .js to text/plain; with
// X-Content-Type-Options: nosniff the browser would then refuse the module
// script (golang/go#32350).
var uiContentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
}

// newUIHandler serves the built single-page app from fsys (rooted at the
// Vite outDir) under uiPrefix. built=false serves a translated page telling
// developers to build the UI. Auth and Host/Origin checks stay in
// secureHandler; this only serves static files.
//
// Cache policy (as in Tailscale's web client, client/web/assets.go): files
// under assets/ have content hashes in their names, so they are immutable;
// index.html must always be revalidated so a new binary's UI loads at once.
func newUIHandler(fsys fs.FS, built bool) http.Handler {
	gz := &gzipCache{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !built {
			serveUINotBuilt(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, uiPrefix)), "/")
		if name == "" {
			name = "index.html"
		}
		if serveUIFile(w, r, fsys, gz, name) {
			return
		}
		// A missing hashed asset is a real 404: answering with HTML would
		// make the browser report a confusing MIME error instead.
		if name == "assets" || strings.HasPrefix(name, "assets/") {
			http.NotFound(w, r)
			return
		}
		// Client-side route (e.g. /app/settings): hand it to the SPA.
		if !serveUIFile(w, r, fsys, gz, "index.html") {
			http.NotFound(w, r)
		}
	})
}

// serveUIFile serves one regular file and reports whether it existed.
// Directories are never listed.
func serveUIFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, gz *gzipCache, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return false
	}
	if ct, ok := uiContentTypes[strings.ToLower(path.Ext(name))]; ok {
		w.Header().Set("Content-Type", ct)
	}
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	if uiCompressible[strings.ToLower(path.Ext(name))] {
		// The answer depends on Accept-Encoding, also for caches.
		w.Header().Add("Vary", "Accept-Encoding")
		if acceptsGzip(r.Header.Get("Accept-Encoding")) {
			if body, ok := gz.get(name, rs); ok {
				w.Header().Set("Content-Encoding", "gzip")
				// Ranges would apply to the gzip bytes; send it whole.
				r.Header.Del("Range")
				http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(body))
				return true
			}
			if _, err := rs.Seek(0, io.SeekStart); err != nil {
				return false
			}
		}
	}
	http.ServeContent(w, r, name, time.Time{}, rs)
	return true
}

// uiCompressible are the text types worth gzip; images, fonts (woff2 is
// already Brotli-compressed) and other binary formats are sent as they are.
var uiCompressible = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".json": true, ".svg": true, ".txt": true,
}

// gzipMinSize: smaller files gain nothing from compression (one packet).
const gzipMinSize = 1024

// gzipCache keeps each embedded file's gzip form, made once on first
// request: the files are embedded in the binary and never change, so the
// cost is paid once per file per daemon run (no brotli: no vetted Brotli
// library is in go.mod; gzip is supported by every browser).
type gzipCache struct {
	mu    sync.Mutex
	files map[string][]byte // nil value = not worth compressing
}

func (c *gzipCache) get(name string, rs io.ReadSeeker) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.files[name]; ok {
		return b, b != nil
	}
	if c.files == nil {
		c.files = map[string][]byte{}
	}
	raw, err := io.ReadAll(rs)
	if err != nil {
		return nil, false // not cached: try again next time
	}
	var out []byte
	if len(raw) >= gzipMinSize {
		var buf bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		_, werr := zw.Write(raw)
		if cerr := zw.Close(); werr == nil && cerr == nil && buf.Len() < len(raw) {
			out = buf.Bytes()
		}
	}
	c.files[name] = out
	return out, out != nil
}

// acceptsGzip parses Accept-Encoding (RFC 9110 §12.5.3): gzip (or *) with
// a q-value above 0.
func acceptsGzip(header string) bool {
	star := false
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		coding = strings.ToLower(strings.TrimSpace(coding))
		q := 1.0
		for _, p := range strings.Split(params, ";") {
			k, v, ok := strings.Cut(strings.TrimSpace(p), "=")
			if ok && strings.EqualFold(strings.TrimSpace(k), "q") {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					q = f
				}
			}
		}
		switch coding {
		case "gzip", "x-gzip":
			return q > 0
		case "*":
			star = q > 0
		}
	}
	return star
}

var uiNotBuiltPage = template.Must(template.New("ui").Parse(`<!doctype html>
<html lang="{{.Lang}}"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark"><title>Kongtrol</title>
<style>body{font:16px/1.55 system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem}
code{font-family:ui-monospace,monospace}</style></head>
<body><h1>{{.Title}}</h1><p>{{.Body}}</p><p>{{.Fix}}</p><p><a href="/">{{.Back}}</a></p></body></html>
`))

// serveUINotBuilt answers 503 so scripts and smoke tests can tell a binary
// built without the UI apart from a working one.
func serveUINotBuilt(w http.ResponseWriter, r *http.Request) {
	lang := requestLang(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	if r.Method == http.MethodHead {
		return
	}
	// A write error only means the client went away.
	_ = uiNotBuiltPage.Execute(w, map[string]string{
		"Lang":  string(lang),
		"Title": i18n.T(lang, "ui.not_built.title"),
		"Body":  i18n.T(lang, "ui.not_built.body"),
		"Fix":   i18n.T(lang, "ui.not_built.fix"),
		"Back":  i18n.T(lang, "ui.not_built.back"),
	})
}
