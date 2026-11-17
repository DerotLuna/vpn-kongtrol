package api

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// brandPrefix is where the brand files embedded from assets/brand (logo
// marks, SVG masters, self-hosted fonts) are served. The legacy dashboard
// links /brand/... instead of keeping its own copies: //go:embed cannot
// reach ../assets from web/, so the files come from package assets.
const brandPrefix = "/brand/"

// brandCacheControl lets the browser keep brand files for 30 days. Their
// names carry no content hash, so they are not marked immutable: a new
// logo reaches existing browsers at the latest a month after an upgrade.
const brandCacheControl = "public, max-age=2592000"

// newBrandHandler serves fsys (rooted at assets/brand) under brandPrefix.
// Content types come from uiContentTypes so the Windows registry cannot
// override them; unknown extensions and directories are 404.
func newBrandHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, brandPrefix)), "/")
		ct, known := uiContentTypes[strings.ToLower(path.Ext(name))]
		if name == "" || !known {
			http.NotFound(w, r)
			return
		}
		f, err := fsys.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		rs, seekable := f.(io.ReadSeeker)
		if err != nil || info.IsDir() || !seekable {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", brandCacheControl)
		http.ServeContent(w, r, name, time.Time{}, rs)
	})
}
