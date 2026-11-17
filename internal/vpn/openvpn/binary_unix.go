//go:build linux || darwin

package openvpn

// platformCandidates returns common install locations that are frequently
// missing from a non-root user's PATH: distro packages install to /usr/sbin,
// Homebrew to /opt/homebrew/sbin (Apple Silicon) or /usr/local/sbin (Intel).
func platformCandidates() []string {
	return []string{
		"/usr/sbin/openvpn",
		"/usr/local/sbin/openvpn",
		"/opt/homebrew/sbin/openvpn",
		"/usr/bin/openvpn",
		"/opt/local/sbin/openvpn", // MacPorts
	}
}
