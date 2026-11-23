//go:build !windows && !linux && !darwin

package autostart

// Status: no per-user sign-in start on this OS.
func Status() State { return State{} }

// Set is not available on this OS.
func Set(bool) error { return ErrNoTray }

// Remove has nothing to remove on this OS.
func Remove(string) error { return nil }
