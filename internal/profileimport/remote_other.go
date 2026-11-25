//go:build !windows && !linux && !darwin

package profileimport

// onRemoteDrive: no file system check on the other Unixes; the search and
// references outside the .ovpn's folder are off there (ADR-0112).
func onRemoteDrive(string) bool { return true }
