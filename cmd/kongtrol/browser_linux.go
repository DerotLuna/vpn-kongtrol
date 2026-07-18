//go:build linux

package main

import "os/exec"

// launchBrowser opens url in the OS default browser on Linux.
func launchBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}
