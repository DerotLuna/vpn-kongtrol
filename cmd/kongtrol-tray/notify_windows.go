//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// toastScript shows a Windows toast. Title and body come from environment
// variables and go through CreateTextNode, so they are never parsed as
// script or XML.
const toastScript = `$m=[Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime];` +
	`$x=$m::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02);` +
	`$t=$x.GetElementsByTagName('text');` +
	`[void]$t.Item(0).AppendChild($x.CreateTextNode($env:KONGTROL_TOAST_TITLE));` +
	`[void]$t.Item(1).AppendChild($x.CreateTextNode($env:KONGTROL_TOAST_BODY));` +
	`$m::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show([Windows.UI.Notifications.ToastNotification]::new($x))`

// desktopNotify shows a desktop notification (best effort).
func desktopNotify(title, body string) {
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", toastScript)
	cmd.Env = append(os.Environ(), "KONGTROL_TOAST_TITLE="+title, "KONGTROL_TOAST_BODY="+body)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("kongtrol-tray: notification: %v", err)
		}
	}()
}
