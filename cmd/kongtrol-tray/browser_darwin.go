//go:build darwin

package main

import "os/exec"

// openBrowser opens url in the user's default browser.
func openBrowser(url string) error {
	return exec.Command("open", url).Start()
}
