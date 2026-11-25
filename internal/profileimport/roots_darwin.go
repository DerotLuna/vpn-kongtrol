//go:build darwin

package profileimport

import "path/filepath"

// SearchRoots are the folders searched for a dropped file's original
// (ADR-0112): Downloads, Desktop, Documents, and the configuration folders
// of Tunnelblick (private configurations, ~/Library/Application
// Support/Tunnelblick/Configurations/<name>.tblk/…) and Viscosity
// (~/Library/Application Support/Viscosity/OpenVPN/<n>/config.conf).
func SearchRoots() []string {
	h := home()
	if h == "" {
		return nil
	}
	support := filepath.Join(h, "Library", "Application Support")
	return existing(
		filepath.Join(h, "Downloads"), filepath.Join(h, "Desktop"), filepath.Join(h, "Documents"),
		filepath.Join(support, "Tunnelblick", "Configurations"),
		filepath.Join(support, "Viscosity", "OpenVPN"),
	)
}
