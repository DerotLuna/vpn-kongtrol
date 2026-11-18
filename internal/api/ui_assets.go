package api

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// The dashboard (web/ui, embedded by web/ui_embed.go) is served at "/"
// (ADR-0043). Until then it lived under legacyUIPrefix, next to the classic
// dashboard; bookmarks and older tray builds still open those addresses, so
// they get a permanent redirect to the same page under "/".
const legacyUIPrefix = "/app"

// legacyPages are the classic dashboard's pages (deleted, ADR-0043) and the
// section that took each one over, so bookmarks and older tray builds (the
// first-run "/studio.html") still land on the right page.
var legacyPages = map[string]string{
	"/studio.html":   "/profiles",
	"/security.html": "/security",
	"/settings.html": "/settings",
	"/audit.html":    "/activity",
}

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
// Vite outDir) at "/", the catch-all of the server mux: every path that no
// API route claimed. built=false serves a translated page telling
// developers to build the UI. Auth and Host/Origin checks stay in
// secureHandler; this only serves static files.
//
// Cache policy (as in Tailscale's web client, client/web/assets.go): files
// under assets/ have content hashes in their names, so they are immutable;
// index.html must always be revalidated so a new binary's UI loads at once.
func newUIHandler(fsys fs.FS, built bool) http.Handler {
	gz := &gzipCache{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// An API path no route matched is an API 404, never the page.
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if target, ok := legacyUIRedirect(r.URL); ok {
			// 308 keeps the method; the browser re-attaches the "#k=…"
			// sign-in fragment, which never reaches the server, because
			// Location carries none (RFC 9110 §10.2.2).
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
			return
		}
		if !built {
			serveUINotBuilt(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if serveUIFile(w, r, fsys, gz, name) {
			return
		}
		// A missing file is a real 404: answering with HTML would make the
		// browser report a confusing MIME error instead. Hashed assets live
		// under assets/; a name with an extension (favicon.ico, robots.txt,
		// an old /studio.html) is a file too. Client routes have neither.
		if name == "assets" || strings.HasPrefix(name, "assets/") || path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		// Client-side route (e.g. /settings): hand it to the SPA.
		if !serveUIFile(w, r, fsys, gz, "index.html") {
			http.NotFound(w, r)
		}
	})
}

// legacyUIRedirect maps an address of the old /app/ prefix to the same page
// at "/" (query kept): /app → /, /app/rules?x=1 → /rules?x=1. Paths that
// only start with the letters (/apple) are not redirected. Classic
// dashboard pages map to their new section (legacyPages).
func legacyUIRedirect(u *url.URL) (string, bool) {
	p := u.Path
	if to, ok := legacyPages[p]; ok {
		return to, true
	}
	if p != legacyUIPrefix && !strings.HasPrefix(p, legacyUIPrefix+"/") {
		return "", false
	}
	target := path.Clean("/" + strings.TrimPrefix(p, legacyUIPrefix))
	// Browsers read "/\host" as "//host" (another site): never redirect
	// there; such a path is no page of ours anyway.
	if strings.ContainsRune(target, '\\') {
		return "/", true
	}
	if u.RawQuery != "" {
		target += "?" + u.RawQuery
	}
	return target, true
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

// uiNotBuiltCSS is the not-built page's only style. The page CSP allows no
// inline style, so this response names the block by its SHA-256 hash (CSP
// Level 3 hash-source) instead of turning on 'unsafe-inline'.
const uiNotBuiltCSS = `body{font:16px/1.55 system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem}`

var uiNotBuiltCSP = func() string {
	sum := sha256.Sum256([]byte(uiNotBuiltCSS))
	return "default-src 'none'; style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'; frame-ancestors 'none'"
}()

var uiNotBuiltPage = template.Must(template.New("ui").Parse(`<!doctype html>
<html lang="{{.Lang}}"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark"><title>Kongtrol</title>
<style>` + uiNotBuiltCSS + `</style></head>
<body><h1>{{.Title}}</h1><p>{{.Body}}</p><p>{{.Fix}}</p></body></html>
`))

// serveUINotBuilt answers 503 so scripts and smoke tests can tell a binary
// built without the UI apart from a working one.
func serveUINotBuilt(w http.ResponseWriter, r *http.Request) {
	lang := requestLang(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", uiNotBuiltCSP)
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
	})
}
