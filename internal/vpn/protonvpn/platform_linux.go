//go:build linux

package protonvpn

// CheckPlatform reports whether this adapter can run on the current OS.
// protonvpn-cli is available on Linux, so nothing to check.
func CheckPlatform() error { return nil }
