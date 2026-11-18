package main

import (
	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/assets"
)

// Linux: systray publishes a StatusNotifierItem IconPixmap (it decodes
// PNG; .ico is not decodable there) and menu icons as dbusmenu
// "icon-data" PNG. There is no portable panel-theme query, so the mark
// uses the light ink of symbolic icons on the usual dark panels (GNOME's
// top bar); the badge carries the state.

const menuIconSize = 16

func menuIconBytes(png []byte) []byte { return png }

func trayIconTheme() string { return "symbolic" }

func applyTrayIcon(badge assets.TrayBadge, _ string) error {
	png, err := assets.TrayPNG(32, assets.InkOnDark, badge)
	if err != nil {
		return err
	}
	systray.SetIcon(png)
	return nil
}
