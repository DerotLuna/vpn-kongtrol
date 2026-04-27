// Package assets embeds the web dashboard static files into the binary.
// The embed directive must live adjacent to the dashboard/ directory.
package assets

import "embed"

//go:embed dashboard
var FS embed.FS
