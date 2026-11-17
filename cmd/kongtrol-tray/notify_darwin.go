//go:build darwin

package main

import (
	"log"
	"os/exec"
)

// desktopNotify shows a notification via osascript (best effort). Title
// and body are script arguments, never part of the script text.
func desktopNotify(title, body string) {
	cmd := exec.Command("/usr/bin/osascript",
		"-e", "on run argv", "-e", "display notification (item 2 of argv) with title (item 1 of argv)", "-e", "end run",
		title, body)
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("kongtrol-tray: notification: %v", err)
		}
	}()
}
