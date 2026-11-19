//go:build !windows

package api

// plainLocalVolume: Unix paths have no volume; cleanImportPath already
// refuses the device trees.
func plainLocalVolume(string) bool { return true }
