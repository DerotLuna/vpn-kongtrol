//go:build !windows

package main

import "context"

// peVersion is a no-op on non-Windows platforms.
func peVersion(_ string) string { return "" }

// fileVersion is a no-op on non-Windows platforms (no PE version resource).
func fileVersion(_ context.Context, _ string) string { return "" }
