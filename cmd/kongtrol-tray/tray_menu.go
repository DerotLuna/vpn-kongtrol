package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/assets"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func onReady() {
	refreshTrayLanguage()
	systray.SetTitle("VPN Kongtrol")
	systray.SetTooltip(trayT("tray.tooltip"))

	// fyne.io/systray races NIM_ADD (inside its own Windows init) against the
	// first SetIcon() call from onReady, which runs on a separate goroutine
	// the moment init finishes — Explorer sometimes hasn't caught up to the
	// freshly-added tray entry yet, so Shell_NotifyIcon(NIM_MODIFY) fails with
	// a misleading "operation completed successfully" error. A short delay
	// avoids the race; it's harmless if unneeded.
	if runtime.GOOS == "windows" {
		time.Sleep(150 * time.Millisecond)
	}
	setIcon(false)

	if err := initDaemon(); err != nil {
		systray.SetTitle(trayT("tray.config_error"))
		systray.SetTooltip(err.Error())
		log.Printf("kongtrol-tray: init: %v", err)

		mQuit := systray.AddMenuItem(trayT("tray.quit"), trayT("tray.exit"))
		go func() {
			<-mQuit.ClickedCh
			systray.Quit()
		}()
		return
	}

	// ── Menu items ────────────────────────────────────────────────────────

	mDashboard := systray.AddMenuItem(trayT("tray.open_dashboard"), trayT("tray.open_dashboard_tip"))
	labels := trayMenuLabels{dashboard: mDashboard}
	onClick(mDashboard, func() { openBrowser(srv.Addr()) })
	systray.AddSeparator()

	// Groups submenu — one entry per named group in kongtrol.yaml. Clicking
	// a group connects every disconnected profile in it, or disconnects all
	// of them if every profile in the group is already connected.
	groupItems := make(map[string]*systray.MenuItem)
	if len(cfg.Groups) > 0 {
		mGroups := systray.AddMenuItem(trayT("tray.groups"), trayT("tray.groups_tip"))
		labels.groups = mGroups
		for name := range cfg.Groups {
			item := mGroups.AddSubMenuItem(name, "")
			groupItems[name] = item
			onClick(item, func() { toggleGroup(name) })
		}
		systray.AddSeparator()
	}

	// Tunnels submenu — one entry per VPN profile.
	mTunnels := systray.AddMenuItem(trayT("tray.tunnels"), trayT("tray.tunnels_tip"))
	labels.tunnels = mTunnels
	profileItems := make(map[string]*systray.MenuItem)
	for name := range cfg.VPNs {
		item := mTunnels.AddSubMenuItem(name, "")
		profileItems[name] = item
		onClick(item, func() { toggleProfile(name) })
	}

	systray.AddSeparator()
	mStatus := systray.AddMenuItem(trayT("tray.status_loading"), "")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(trayT("tray.quit"), trayT("tray.quit_tip"))
	labels.quit = mQuit
	onClick(mQuit, systray.Quit)

	// ── Background: status refresh ────────────────────────────────────────

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateTrayStatus(mStatus, profileItems, groupItems, labels)
		}
	}()
}

// onClick spawns a dedicated goroutine that blocks on item.ClickedCh and
// runs fn on every click.
//
// systray's Windows/Linux/macOS backends all deliver clicks with a
// non-blocking channel send (`select { case ch <- struct{}{}: default: }`,
// see systrayMenuItemSelected in the vendored library) — if nothing is
// actively receiving on ClickedCh at that exact instant, the click is
// dropped, not queued. A shared loop that polls many items with short
// sleeps in between (the previous approach here) is blocked most of the
// time, so most clicks were silently lost. One goroutine per item blocked
// in `for range item.ClickedCh` is the only way to reliably catch every
// click, and matches the pattern from systray's own examples.
func onClick(item *systray.MenuItem, fn func()) {
	go func() {
		for range item.ClickedCh {
			fn()
		}
	}()
}

func updateTrayStatus(
	mStatus *systray.MenuItem,
	profileItems, groupItems map[string]*systray.MenuItem,
	labels trayMenuLabels,
) {
	snapshot := col.Snapshot()
	connectedCount := 0
	for _, m := range snapshot {
		if m.Status == vpn.StatusConnected {
			connectedCount++
		}
	}

	if refreshTrayLanguage() {
		applyTrayMenuLanguage(labels)
	}
	label := trayF("tray.status_active", connectedCount)
	mStatus.SetTitle(label)
	setIcon(connectedCount > 0)

	for name, item := range profileItems {
		if m, ok := snapshot[name]; ok {
			item.SetTitle(fmt.Sprintf("%s [%s]", name, m.Status))
		}
	}

	for name, item := range groupItems {
		g, ok := cfg.Groups[name]
		if !ok {
			continue
		}
		connected := 0
		for _, p := range g.Profiles {
			if m, ok := snapshot[p]; ok && m.Status == vpn.StatusConnected {
				connected++
			}
		}

		item.SetTitle(fmt.Sprintf("%s (%d/%d)", name, connected, len(g.Profiles)))
	}
}

func refreshTrayLanguage() bool {
	path, err := config.PreferencesPath("")
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || (!trayPrefsMod.IsZero() && info.ModTime().Equal(trayPrefsMod)) {
		return false
	}
	prefs, err := config.LoadPreferences(path)
	if err != nil {
		return false
	}
	next := i18n.ES
	if prefs.Language == "en" {
		next = i18n.EN
	}
	changed := next != trayLang
	trayLang = next
	trayPrefsMod = info.ModTime()
	return changed
}

func applyTrayMenuLanguage(labels trayMenuLabels) {
	systray.SetTooltip(trayT("tray.tooltip"))
	if labels.dashboard != nil {
		labels.dashboard.SetTitle(trayT("tray.open_dashboard"))
		labels.dashboard.SetTooltip(trayT("tray.open_dashboard_tip"))
	}
	if labels.groups != nil {
		labels.groups.SetTitle(trayT("tray.groups"))
		labels.groups.SetTooltip(trayT("tray.groups_tip"))
	}
	if labels.tunnels != nil {
		labels.tunnels.SetTitle(trayT("tray.tunnels"))
		labels.tunnels.SetTooltip(trayT("tray.tunnels_tip"))
	}
	if labels.quit != nil {
		labels.quit.SetTitle(trayT("tray.quit"))
		labels.quit.SetTooltip(trayT("tray.quit_tip"))
	}
}

func applyTrayDNSGuardState() {
	if dnsMgr == nil || cfg == nil {
		return
	}
	if !cfg.Security.DNSGuard.Enabled {
		dnsMgr.ForceRestore()
		return
	}
	for name, adapter := range adapters {
		vpnCfg, ok := cfg.VPNs[name]
		if !ok || vpnCfg.Type == "wireguard" || adapter.Status().Normalize() != vpn.StatusConnected {
			continue
		}
		info, err := adapter.TunnelInfo()
		if err != nil || info == nil {
			continue
		}
		dnsServers := info.DNS
		if len(dnsServers) == 0 && cfg.Security.DNSGuard.FallbackDNS != "" {
			if ip := net.ParseIP(cfg.Security.DNSGuard.FallbackDNS); ip != nil {
				dnsServers = []net.IP{ip}
			}
		}
		if len(dnsServers) > 0 {
			dnsMgr.OnConnect(name, info.InterfaceName, dnsServers)
		}
	}
}

// toggleProfile connects a disconnected profile (fetching credentials from
// the OS keychain via ProfileService.Connect, exactly like `kongtrol up`)
// or disconnects a connected one.

func setIcon(connected bool) {
	size := 32
	iconBytes, err := assets.TrayIcon(size)
	if err != nil {
		// Fallback: use the full-res PNG and let systray resize.
		iconBytes = assets.LogoKongPNG
	}
	systray.SetIcon(iconBytes)
	_ = connected // future: swap to a grey version when disconnected
}
