//go:build linux

package profileimport

import "golang.org/x/sys/unix"

// Network and user-space file systems (statfs f_type, linux/magic.h): NFS,
// SMB/CIFS, SMB2, AFS, Ceph, 9P and FUSE (sshfs, rclone and other remote
// mounts are FUSE; a local FUSE disk is skipped too, on the safe side).
var remoteFS = map[int64]bool{
	0x6969: true, 0x517B: true, 0xFF534D42: true, 0xFE534D42: true,
	0x5346414F: true, 0x00C36400: true, 0x01021997: true, 0x65735546: true,
}

// onRemoteDrive: p is on a network (or FUSE) file system, or its file
// system cannot be told. Such a file is never opened by the search or for
// a reference: reading it may contact another machine.
func onRemoteDrive(p string) bool {
	var st unix.Statfs_t
	if err := unix.Statfs(p, &st); err != nil {
		return true
	}
	return remoteFS[int64(st.Type)] //nolint:unconvert // Type is int32 on some arches
}
