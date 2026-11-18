//go:build !windows

package main

import (
	"log"
	"runtime"
)

// setupFlyout keeps the native menu on the left click outside Windows
// (ADR-0036): macOS would need an NSPopover anchored to fyne.io/systray's
// status item, which the library does not expose, and Linux trays
// (StatusNotifierItem) do not report reliably where the icon is.
func setupFlyout() {
	_, reason := trayLeftClick(runtime.GOOS, false)
	log.Printf("kongtrol-tray: %s", reason)
}
