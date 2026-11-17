//go:build linux

package main

import (
	"log"
	"os/exec"
)

// desktopNotify shows a notification via notify-send when installed (best
// effort).
func desktopNotify(title, body string) {
	p, err := exec.LookPath("notify-send")
	if err != nil {
		return
	}
	cmd := exec.Command(p, "--app-name=VPN Kongtrol", title, body)
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("kongtrol-tray: notification: %v", err)
		}
	}()
}
