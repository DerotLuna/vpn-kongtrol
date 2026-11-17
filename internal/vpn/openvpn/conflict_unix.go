//go:build linux || darwin

package openvpn

// openvpnCommandLines returns the command lines of all processes; the
// caller filters for openvpn.
func openvpnCommandLines() string {
	out, _ := runCmd("ps", "-axo", "command")
	return out
}
