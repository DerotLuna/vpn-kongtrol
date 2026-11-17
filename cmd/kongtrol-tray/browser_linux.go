//go:build linux

package main

import "os/exec"

// openBrowser opens url in the user's default browser.
func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}
