//go:build windows

package main

import "context"

// readProtectedElevated: Windows profile files are not root-only the way
// /etc/wireguard is; the user is told how to copy the file instead.
func readProtectedElevated(context.Context, string) ([]byte, error) {
	return nil, errElevatedReadUnavailable
}

// apiProtectedReader: none on Windows.
var apiProtectedReader func(context.Context, string) ([]byte, error)
