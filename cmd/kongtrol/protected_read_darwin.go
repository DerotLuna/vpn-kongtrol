//go:build darwin

package main

import "context"

// elevatedCapture: macOS has no pkexec; without a terminal the user is
// told how to copy the file instead (the "what to do" line).
func elevatedCapture(context.Context, []string) ([]byte, error) {
	return nil, errElevatedReadUnavailable
}

// apiProtectedReader: none without a terminal on macOS.
var apiProtectedReader func(context.Context, string) ([]byte, error)
