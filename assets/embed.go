// Package assets embeds the brand assets from assets/brand/ (the single
// committed copy of the logo, marks and self-hosted fonts) for the tray app,
// the CLI and the dashboard HTTP server.
package assets

import (
	"embed"
	"io/fs"
)

// LogoPNG is the full logo with "VPN Kongtrol" wordmark (512×512) — use for
// splash screens, about dialogs, and other contexts where the lockup fits.
//
//go:embed brand/logo-512.png
var LogoPNG []byte

// LogoKongPNG is the icon-only mark, no text (512×512) — use for tray icons,
// favicons, and any context smaller than ~128px where the wordmark would be
// unreadable. TrayIcon downscales it.
//
//go:embed brand/logo-kong-512.png
var LogoKongPNG []byte

// brandFiles holds what the dashboard serves under /brand/: the SVG masters,
// the PNG rasters and the self-hosted web fonts. generate.go, source/ masters
// and the Open Graph JPEG are deliberately left out of the binary.
//
//go:embed brand/*.svg brand/*.png brand/fonts
var brandFiles embed.FS

// Brand returns the embedded brand files rooted at assets/brand, e.g.
// "logo-kong.svg" or "fonts/fonts.css".
func Brand() fs.FS {
	sub, err := fs.Sub(brandFiles, "brand")
	if err != nil {
		panic("assets: brand embed: " + err.Error()) // static pattern; cannot fail
	}
	return sub
}
