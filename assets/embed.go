// Package assets embeds the application logo and other static resources
// for use in the tray app and CLI (favicon, icon bytes, etc.).
package assets

import (
	_ "embed"
)

// LogoPNG is the full logo with "VPN Kongtrol" wordmark — use for splash
// screens, about dialogs, and other contexts where the full brand lockup fits.
//go:embed logo.png
var LogoPNG []byte

// LogoKongPNG is the icon-only mark (no text) — use for tray icons, favicons,
// and any context smaller than ~128px where the wordmark would be unreadable.
//go:embed logo-kong.png
var LogoKongPNG []byte
