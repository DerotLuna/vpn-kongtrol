package main

import (
	"sync"

	"fyne.io/systray"
)

var (
	publicIPMu   sync.Mutex
	publicIPItem *systray.MenuItem
)

// addPublicIPMenuItem creates "IP pública: 1.2.3.4 · Revisar". It is hidden
// while the opt-in public IP lookup is off; a click asks the daemon for one
// lookup now.
func addPublicIPMenuItem() {
	item := systray.AddMenuItem("", trayT("tray.public_ip_tip"))
	item.Hide()
	publicIPMu.Lock()
	publicIPItem = item
	publicIPMu.Unlock()
	onClick(item, func() {
		if _, err := client.CheckPublicIP(); err != nil {
			notifyError(trayT("cli.check.failed_hint"), err)
		}
		refreshPublicIPItem()
	})
}

// refreshPublicIPItem shows the daemon's last lookup (no external request).
func refreshPublicIPItem() {
	publicIPMu.Lock()
	item := publicIPItem
	publicIPMu.Unlock()
	if item == nil {
		return
	}
	st, err := client.PublicIP()
	if err != nil || !st.Enabled {
		item.Hide()
		return
	}
	ip := "—"
	if st.Check != nil && st.Check.PublicIP != "" {
		ip = st.Check.PublicIP
	}
	item.SetTitle(trayF("tray.public_ip", ip))
	item.SetTooltip(trayT("tray.public_ip_tip"))
	item.Show()
}
