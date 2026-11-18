package main

import (
	"fyne.io/systray"
	"golang.org/x/sys/windows/registry"

	"github.com/vpn-kongtrol/kongtrol/assets"
)

// Windows: the tray (notification area) loads icons with
// LoadImage(IMAGE_ICON) from a file, so icons are .ico. The mark is tinted
// for the taskbar theme: HKCU\Software\Microsoft\Windows\CurrentVersion\
// Themes\Personalize\SystemUsesLightTheme (1 = light taskbar/Start, 0 or
// missing = dark, Windows 11's default). It is re-read on every refresh,
// so switching the theme updates the icon within seconds.

// menuIconSize: systray converts menu icons to a SM_CXSMICON bitmap; it
// loads the .ico at the default (large) size first, so 32px scales down.
const menuIconSize = 32

func menuIconBytes(png []byte) []byte { return assets.MenuIconICO(png, menuIconSize) }

// taskbarIsLight reports the taskbar theme (false when unknown).
func taskbarIsLight() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("SystemUsesLightTheme")
	return err == nil && v == 1
}

// trayIconTheme is part of the icon cache key: the icon changes with it.
func trayIconTheme() string {
	if taskbarIsLight() {
		return "light"
	}
	return "dark"
}

func applyTrayIcon(badge assets.TrayBadge, theme string) error {
	ink := assets.InkOnDark
	if theme == "light" {
		ink = assets.InkOnLight
	}
	ico, err := assets.TrayICO(ink, badge)
	if err != nil {
		return err
	}
	systray.SetIcon(ico)
	return nil
}
