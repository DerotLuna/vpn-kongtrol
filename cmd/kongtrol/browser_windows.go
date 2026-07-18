//go:build windows

package main

import "os/exec"

// launchBrowser opens url in the OS default browser on Windows.
func launchBrowser(url string) error {
	// The empty argument is required: cmd's "start" treats the first
	// quoted argument as the new window's title, not the target.
	return exec.Command("cmd", "/c", "start", "", url).Start()
}
