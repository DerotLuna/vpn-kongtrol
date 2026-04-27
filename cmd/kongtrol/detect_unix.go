//go:build !windows

package main

// peVersion is a no-op on non-Windows platforms.
func peVersion(_ string) string { return "" }
