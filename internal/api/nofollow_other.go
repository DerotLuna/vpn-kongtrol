//go:build !unix

package api

// openNoFollow: Windows has no open flag for it; readRegularFile compares
// the opened file with the Lstat result (os.SameFile) instead.
const openNoFollow = 0
