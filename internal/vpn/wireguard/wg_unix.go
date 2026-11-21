//go:build darwin

package wireguard

// wgPreUp leaves macOS tunnel setup to wg-quick in the helper.
func wgPreUp(string) bool { return false }

// tunnelServiceRunning is a no-op on Unix; WireGuard is managed by wg-quick,
// not a Windows service. Always returns false.
func tunnelServiceRunning(_ string) bool { return false }
