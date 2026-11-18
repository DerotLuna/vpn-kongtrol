package main

import (
	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/assets"
)

// macOS: the menu bar icon is a template image (systray.SetTemplateIcon →
// NSImage.template), so macOS tints it for the menu bar's light/dark look.
// Only the alpha channel counts: the state badge stays as a shape (filled
// dot = connected/caution, ring = stopped); the menu header says the word.
// systray draws it at 16×16 pt, so the 32px mask is the @2x rendition.

const menuIconSize = 32 // drawn at 16 pt by systray

func menuIconBytes(png []byte) []byte { return png }

func trayIconTheme() string { return "template" }

func applyTrayIcon(badge assets.TrayBadge, _ string) error {
	png, err := assets.TrayPNG(32, assets.InkTemplate, badge)
	if err != nil {
		return err
	}
	systray.SetTemplateIcon(png, png)
	return nil
}
