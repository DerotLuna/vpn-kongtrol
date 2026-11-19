//go:build !windows

package main

// Only Windows opens a console window for a double-clicked program.
func startedByExplorer() bool { return false }
