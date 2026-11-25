//go:build !windows

package profileimport

import "io/fs"

// plainLocalVolume: Unix paths have no volume; CleanLocalPath already
// refuses the device trees.
func plainLocalVolume(string) bool { return true }

// plain says whether fi (from Lstat) is a real folder or file: no link, no
// FIFO, socket or device.
func plain(_ string, fi fs.FileInfo) bool {
	return fi.Mode()&(fs.ModeSymlink|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeDevice|fs.ModeCharDevice|fs.ModeIrregular) == 0
}
