//go:build darwin

package main

import "os/exec"

// launchBrowser opens url in the OS default browser on macOS.
func launchBrowser(url string) error {
	return exec.Command("open", url).Start()
}
