//go:build !windows

package main

// acquireSingleInstance is a no-op on platforms without a lighter-weight
// equivalent wired up yet; always allows startup.
func acquireSingleInstance() bool { return true }
