//go:build linux || darwin

package wireguard

// wgPreUp: nothing to check on Unix; wg-quick runs in the helper.
func wgPreUp(string) bool { return false }

// wgShow returns the output of `wg show <iface>`. It needs root, so it
// fails for the unprivileged daemon and Status() falls back to the
// interface check.
func wgShow(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName)
}

// wgTransfer returns byte counts: `wg show <iface> transfer`.
// Output: "<peer-pubkey>\t<rx-bytes>\t<tx-bytes>"
func wgTransfer(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName, "transfer")
}

// tunnelServiceRunning is a no-op on Unix; WireGuard is managed by wg-quick,
// not a Windows service. Always returns false.
func tunnelServiceRunning(_ string) bool { return false }
