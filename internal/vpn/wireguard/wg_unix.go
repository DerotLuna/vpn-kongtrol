//go:build linux || darwin

package wireguard

// wgPreUp: nothing to check on Unix; wg-quick runs in the helper.
func wgPreUp(string) bool { return false }

// tunnelServiceRunning is a no-op on Unix; WireGuard is managed by wg-quick,
// not a Windows service. Always returns false.
func tunnelServiceRunning(_ string) bool { return false }
