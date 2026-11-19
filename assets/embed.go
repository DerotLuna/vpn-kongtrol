// Package assets embeds the brand assets from assets/brand/ (the single
// committed copy of the logo, marks and self-hosted fonts) for
// the tray app and the CLI. The dashboard (web/ui) bundles what it needs at
// build time (Vite alias @brand), so the server embeds no copy of its own.
package assets

import (
	_ "embed"
)

// The tray / menu bar icon is not here: icon.go renders it from the
// tray-<px>.png masks (rasterized from the vector mark by generate.go).
