//go:build windows

package openvpn

// openvpnCommandLines returns one command line per running openvpn.exe.
// OpenVPN GUI's tunnels are started by the interactive service under the
// user's own token, so their command lines are readable without elevation.
func openvpnCommandLines() string {
	out, _ := runCmd("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`Get-CimInstance Win32_Process -Filter "Name='openvpn.exe'" | ForEach-Object { $_.CommandLine }`)
	return out
}
