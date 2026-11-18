//go:build !windows

package vpndetect

// FileVersion is "" outside Windows: there is no PE version resource.
func FileVersion(string) string { return "" }
