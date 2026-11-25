//go:build windows

package profileimport

import (
	"io/fs"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// plainLocalVolume: on Windows only a drive-letter path (C:\…) is a local
// path; UNC shares, \\?\ and \??\ forms, device namespaces and volume GUID
// paths are refused (an SMB path would make Windows send the user's
// credentials to that host).
func plainLocalVolume(p string) bool {
	v := filepath.VolumeName(p)
	if len(v) != 2 || v[1] != ':' {
		return false
	}
	c := v[0] | 0x20 // lower case
	return c >= 'a' && c <= 'z'
}

// onRemoteDrive: the drive letter is a mapped network drive
// (GetDriveTypeW = DRIVE_REMOTE). Such a file is never opened by the
// search or for a reference: reading it contacts another machine.
func onRemoteDrive(p string) bool {
	v := filepath.VolumeName(p)
	if v == "" {
		return true
	}
	root, err := windows.UTF16PtrFromString(v + `\`)
	if err != nil {
		return true
	}
	return windows.GetDriveType(root) == windows.DRIVE_REMOTE
}

// ioReparseTagCloud is IO_REPARSE_TAG_CLOUD; its 16 variants differ only in
// bits 12-15 (winnt.h IO_REPARSE_TAG_CLOUD_1 … _F).
const ioReparseTagCloud = 0x9000001A

// plain says whether fi (from Lstat of p) is a real folder or file: no
// link, no junction, no device. Go marks every reparse point that is not a
// link as ModeIrregular; the one kind accepted is OneDrive's cloud files
// and folders (Files On-Demand), which keep their data at that path (a
// Desktop or Documents moved to OneDrive). The tag is read without opening
// the data (FILE_FLAG_OPEN_REPARSE_POINT, attributes only), so a
// placeholder is not downloaded by this check.
func plain(p string, fi fs.FileInfo) bool {
	m := fi.Mode()
	if m&(fs.ModeSymlink|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeDevice|fs.ModeCharDevice) != 0 {
		return false
	}
	if m&fs.ModeIrregular == 0 {
		return true
	}
	tag, ok := reparseTag(p)
	return ok && tag&0xFFFF0FFF == ioReparseTagCloud
}

func reparseTag(p string) (uint32, bool) {
	name, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return 0, false
	}
	h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(h) //nolint:errcheck // read-only handle
	var info struct {
		FileAttributes uint32
		ReparseTag     uint32
	} // FILE_ATTRIBUTE_TAG_INFO
	if err := windows.GetFileInformationByHandleEx(h, windows.FileAttributeTagInfo,
		(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return 0, false
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return 0, true
	}
	return info.ReparseTag, true
}
