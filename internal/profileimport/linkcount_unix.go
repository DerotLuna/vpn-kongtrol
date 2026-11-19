//go:build unix

package profileimport

import (
	"os"
	"syscall"
)

// linkCount returns the number of hard links of fi (from fstat); known is
// false when the platform does not report it.
func linkCount(fi os.FileInfo) (n uint64, known bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Nlink), true //nolint:unconvert // Nlink is uint16 on darwin, uint64 on linux
}
