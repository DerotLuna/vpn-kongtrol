package security

// KillSwitch blocks all network traffic when a VPN tunnel drops unexpectedly.
// Platform-specific implementations are in killswitch_windows.go, _linux.go, _darwin.go.
type KillSwitch interface {
	// Enable activates the kill switch, blocking all traffic except through
	// the tunnel interface. If allowLAN is true, local subnet traffic is permitted.
	Enable(tunnelInterface string, allowLAN bool) error

	// Disable removes all kill switch firewall rules.
	// Always attempt this on shutdown even if Enable was never called.
	Disable() error

	// IsEnabled reports whether the kill switch is currently active.
	IsEnabled() bool
}
