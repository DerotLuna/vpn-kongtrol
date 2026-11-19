//go:build !unix

package profileimport

import "os"

// linkCount is not checked here: the elevated read only runs on Linux and
// macOS (Windows has no root-only profile files, ADR-0058).
func linkCount(os.FileInfo) (n uint64, known bool) { return 0, false }
