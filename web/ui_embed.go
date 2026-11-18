// Package assets embeds the dashboard build (web/ui) into the binary. The
// embed directive must live next to the ui/ directory (no "../" in
// //go:embed paths).
package assets

import (
	"embed"
	"io/fs"
)

// uiDist is the dashboard build (web/ui, Vite + React + TypeScript), served
// at "/" by internal/api (ADR-0043).
//
// web/ui/dist is git-ignored except for dist/.gitkeep, so this pattern always
// matches and `go build` works on a fresh clone without Node; the binary then
// serves a "UI not built" page at "/". `make ui-build` (or `make build`,
// which rebuilds the UI only when its sources changed) fills dist, and
// release.yml always builds it before GoReleaser.
//
//go:embed all:ui/dist
var uiDist embed.FS

// UI returns the built UI rooted at web/ui/dist and reports whether a build
// is present (dist/index.html exists).
func UI() (fs.FS, bool) {
	sub, err := fs.Sub(uiDist, "ui/dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return sub, false
	}
	return sub, true
}
