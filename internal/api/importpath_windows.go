//go:build windows

package api

import "path/filepath"

// plainLocalVolume: on Windows only a drive-letter path (C:\…) may be
// imported; UNC shares, \\?\ and \??\ forms, device namespaces and
// volume GUID paths are refused (an SMB path would make Windows send the
// user's credentials to that host).
func plainLocalVolume(p string) bool {
	v := filepath.VolumeName(p)
	if len(v) != 2 || v[1] != ':' {
		return false
	}
	c := v[0] | 0x20 // lower case
	return c >= 'a' && c <= 'z'
}
