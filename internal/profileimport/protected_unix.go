//go:build unix

package profileimport

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// openNoFollow opens path read-only without following a final symlink
// (O_NOFOLLOW: ELOOP) and without blocking (O_NONBLOCK): a FIFO or device
// swapped in after the Lstat check cannot hang the root process, because
// the caller fstats and refuses anything but a regular file before reading.
func openNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

// linkCount returns the number of hard links of fi (from fstat); known is
// false when the platform does not report it.
func linkCount(fi os.FileInfo) (n uint64, known bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Nlink), true //nolint:unconvert // Nlink is uint16 on darwin, uint64 on linux
}
