// Package assets embeds the application logo and other static resources
// for use in the tray app and CLI (favicon, icon bytes, etc.).
package assets

import (
	_ "embed"
)

// LogoPNG is the full-resolution 1024×1024 application logo (PNG).
//go:embed logo.png
var LogoPNG []byte
