//go:build !unix

package profileimport

import "os"

// The elevated read only runs on Linux and macOS (Windows has no root-only
// profile files, ADR-0058); here it keeps the plain open and no link count.

func openNoFollow(path string) (*os.File, error) { return os.Open(path) }

func openRootNoFollow(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}

func linkCount(os.FileInfo) (n uint64, known bool) { return 0, false }
