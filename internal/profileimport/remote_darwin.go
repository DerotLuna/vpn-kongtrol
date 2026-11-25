//go:build darwin

package profileimport

import "golang.org/x/sys/unix"

// Network file systems by statfs f_fstypename.
var remoteFS = map[string]bool{"smbfs": true, "nfs": true, "afpfs": true, "webdav": true, "cifs": true, "macfuse": true, "osxfuse": true}

// onRemoteDrive: p is on a network file system, or its file system cannot
// be told. Such a file is never opened by the search or for a reference.
func onRemoteDrive(p string) bool {
	var st unix.Statfs_t
	if err := unix.Statfs(p, &st); err != nil {
		return true
	}
	return remoteFS[unix.ByteSliceToString(st.Fstypename[:])]
}
