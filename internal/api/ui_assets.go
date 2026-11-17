package api

import (
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
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

// setControlCookie gives a page served by this daemon (legacy dashboard or
// the new UI) the API capability as an HttpOnly, SameSite=Strict cookie, so
// page scripts never see the token.
func (s *Server) setControlCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "kongtrol_control",
		Value:    s.apiToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
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
		if serveUIFile(w, r, fsys, name) {
			return
		}
		// A missing hashed asset is a real 404: answering with HTML would
		// make the browser report a confusing MIME error instead.
		if name == "assets" || strings.HasPrefix(name, "assets/") {
			http.NotFound(w, r)
			return
		}
		// Client-side route (e.g. /app/settings): hand it to the SPA.
		if !serveUIFile(w, r, fsys, "index.html") {
			http.NotFound(w, r)
		}
	})
}

// serveUIFile serves one regular file and reports whether it existed.
// Directories are never listed.
func serveUIFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) bool {
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
	http.ServeContent(w, r, name, time.Time{}, rs)
	return true
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
