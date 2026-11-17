//go:build !linux

package protonvpn

// CheckPlatform reports whether a Proton CLI can exist on the current OS.
// Proton ships no CLI on Windows or macOS: attach mode opens the Proton VPN
// app and waits for the user (or use `type: wireguard` with a .conf).
func CheckPlatform() error { return ErrUnsupportedOS }
