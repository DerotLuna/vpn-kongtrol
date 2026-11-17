//go:build !windows

package appattach

// osInterfaces lists interfaces by name (Linux / macOS names such as
// nordlynx, proton0, fortissl0 or utun4 carry no description).
func osInterfaces() []Iface { return netInterfaces() }
