//go:build windows

package profileimport

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// SearchRoots are the folders searched for a dropped file's original
// (ADR-0112), most likely first: OpenVPN GUI's config folders (per user,
// then the shared one), then Downloads, Desktop and Documents from the
// Known Folder API (SHGetKnownFolderPath), which follows a folder moved
// elsewhere, such as into OneDrive.
func SearchRoots() []string {
	h := home()
	roots := []string{}
	if h != "" {
		roots = append(roots, filepath.Join(h, "OpenVPN", "config"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		roots = append(roots, filepath.Join(pf, "OpenVPN", "config"))
	}
	for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_Downloads, windows.FOLDERID_Desktop, windows.FOLDERID_Documents} {
		if p, err := windows.KnownFolderPath(id, windows.KF_FLAG_DONT_VERIFY); err == nil {
			roots = append(roots, p)
		}
	}
	return existing(roots...)
}
