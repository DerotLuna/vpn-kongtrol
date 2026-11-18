package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/assets"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
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
	setupMenu()
}

// setupMenu builds the full menu, or the first-run / error menu when the
// tray cannot work yet. It runs again (after systray.ResetMenu) once a
// first-run user has created the config.
func setupMenu() {
	if err := initClient(); err != nil {
		switch {
		case errors.Is(err, errNoConfig):
			showFirstRun()
		case cfg != nil:
			showFatal(trayT("tray.daemon_error"), trayT("tray.daemon_error_hint"), err)
		case kongtrolExe == "":
			showFatal(trayT("tray.cli_missing"), trayT("tray.cli_missing_hint"), err)
		default:
			showFatal(trayT("tray.config_error"), trayT("tray.config_error_hint"), err)
		}
		return
	}

	// ── Menu items ────────────────────────────────────────────────────────

	mDashboard := systray.AddMenuItem(trayT("tray.open_dashboard"), trayT("tray.open_dashboard_tip"))
	labels := trayMenuLabels{dashboard: mDashboard}
	onClick(mDashboard, openDashboard)
	// Shown only while the daemon reports the privileged helper missing
	// (see refreshHelperItem).
	mHelper := systray.AddMenuItem(trayT("tray.helper_install"), trayT("tray.helper_install_tip"))
	mHelper.Hide()
	helperItem.Store(mHelper)
	onClick(mHelper, installHelper)
	systray.AddSeparator()

	// Groups submenu — one entry per named group in kongtrol.yaml. Clicking
	// a group connects every disconnected profile in it, or disconnects all
	// of them if every profile in the group is already connected.
	groupItems := make(map[string]*systray.MenuItem)
	if len(cfg.Groups) > 0 {
		mGroups := systray.AddMenuItem(trayT("tray.groups"), trayT("tray.groups_tip"))
		labels.groups = mGroups
		for _, name := range sortedKeys(cfg.Groups) {
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
	for _, name := range sortedKeys(cfg.VPNs) {
		item := mTunnels.AddSubMenuItem(name, "")
		profileItems[name] = item
		onClick(item, func() { toggleProfile(name) })
	}

	systray.AddSeparator()
	mStatus := systray.AddMenuItem(trayT("tray.status_loading"), "")
	mStatus.Disable()
	noticeMu.Lock()
	statusItem = mStatus
	noticeMu.Unlock()
	// "Conectar <perfil>" after a policy notice (see tray_policy.go).
	addPolicyMenuItem()
	addKillSwitchMenuItem()
	addPublicIPMenuItem()
	systray.AddSeparator()
	mStopAll := systray.AddMenuItem(trayT("tray.stop_all"), trayT("tray.stop_all_tip"))
	labels.stopAll = mStopAll
	onClick(mStopAll, stopKongtrol)
	mQuit := systray.AddMenuItem(trayT("tray.quit"), trayT("tray.quit_tip"))
	labels.quit = mQuit
	// Quitting closes only the tray; the daemon keeps its tunnels up.
	onClick(mQuit, systray.Quit)

	// ── Background: daemon start + status refresh ─────────────────────────

	go func() {
		if err := ensureDaemon(); err != nil {
			notifyError(trayT("tray.err_start"), err)
		}
		updateTrayStatus(mStatus, profileItems, groupItems, labels)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateTrayStatus(mStatus, profileItems, groupItems, labels)
		}
	}()
}

// showFatal leaves a tray with only an error title, a short translated
// "what to do" line and a close item, for when the tray cannot work at all
// (bad config, kongtrol CLI missing). The raw error only goes to the
// line's tooltip and tray.log.
func showFatal(title, hint string, err error) {
	systray.SetTitle(title)
	systray.SetTooltip(title + " · " + hint)
	log.Printf("kongtrol-tray: init: %v", err)

	mErr := systray.AddMenuItem(hint, err.Error())
	mErr.Disable()
	mQuit := systray.AddMenuItem(trayT("tray.exit"), trayT("tray.exit"))
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

// showFirstRun is the menu before any config exists: a translated "set
// it up" line, "Set up now…" (starts the daemon in setup mode and opens the
// dashboard's profile editor, no terminal needed) and a close item. The
// dashboard is also opened once on its own, so a fresh install lands on
// the setup page. When the config appears the full menu replaces this one;
// `kongtrol init` stays the terminal path to the same result.
// firstRunMarker records that the setup page was opened automatically, so a
// user who postpones the setup gets the browser once, not at every login
// (the tray autostarts). The menu item stays available.
func firstRunMarker() string {
	return filepath.Join(daemonctl.StateDir(), "run", "setup-opened")
}

// claimFirstRunOpen reports whether this is the first automatic opening
// and records it. Without a writable state dir it opens nothing on its own.
func claimFirstRunOpen() bool {
	path := firstRunMarker()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func showFirstRun() {
	msg := trayT("tray.first_run")
	systray.SetTooltip(msg)
	log.Println("kongtrol-tray: no config file found; offering setup in the dashboard")

	mMsg := systray.AddMenuItem(msg, trayT("tray.first_run_tip"))
	mMsg.Disable()
	mSetup := systray.AddMenuItem(trayT("tray.setup_open"), trayT("tray.setup_open_tip"))
	openSetup := func() {
		if !openDashboardPage(setupPagePath) {
			mMsg.SetTitle(trayT("tray.setup_failed"))
			systray.SetTooltip(trayT("tray.setup_failed"))
			return
		}
		mMsg.SetTitle(msg)
		systray.SetTooltip(msg)
	}
	onClick(mSetup, openSetup)
	if claimFirstRunOpen() {
		go openSetup()
	}
	mQuit := systray.AddMenuItem(trayT("tray.exit"), trayT("tray.exit"))
	done := make(chan struct{})
	go func() {
		<-mQuit.ClickedCh
		close(done)
		systray.Quit()
	}()

	// Rebuild the full menu as soon as the dashboard (or `kongtrol init`)
	// saved the config.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if configExists() {
					systray.ResetMenu()
					setupMenu()
					return
				}
			}
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
// dropped, not queued. One goroutine per item blocked in
// `for range item.ClickedCh` is the only way to reliably catch every
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
	if refreshTrayLanguage() {
		applyTrayMenuLanguage(labels)
	}

	notice := activeNotice()
	states, err := client.TunnelStates()
	if err != nil {
		if notice != "" {
			mStatus.SetTitle(notice)
			systray.SetTooltip(notice)
		} else {
			mStatus.SetTitle(trayT("tray.daemon_unreachable"))
			systray.SetTooltip(trayT("tray.daemon_unreachable"))
		}
		setIcon(false)
		for name, item := range profileItems {
			item.SetTitle(name)
		}
		return
	}
	systray.SetTooltip(trayT("tray.tooltip"))
	refreshHelperItem()
	pollPolicyEvents(states)
	refreshKillSwitchItem()
	refreshPublicIPItem()
	notice = activeNotice() // a new policy notice shows at once

	connectedCount := 0
	for _, m := range states {
		if m.Status == vpn.StatusConnected {
			connectedCount++
		}
	}
	if notice != "" {
		mStatus.SetTitle(notice)
	} else {
		mStatus.SetTitle(trayF("tray.status_active", connectedCount))
	}
	setIcon(connectedCount > 0)

	names := make([]string, 0, len(profileItems))
	for name, item := range profileItems {
		m, ok := states[name]
		if !ok {
			continue
		}
		names = append(names, name)
		title, tooltip := trayTunnelLabel(name, m)
		item.SetTitle(title)
		item.SetTooltip(tooltip)
	}
	// A provider waiting for the user in its app (or dropped) is the most
	// useful thing the icon tooltip can say.
	sort.Strings(names)
	if notice != "" {
		systray.SetTooltip(notice)
	} else if tip := trayWaitingTooltip(states, names); tip != "" {
		systray.SetTooltip(tip)
	}

	for name, item := range groupItems {
		g, ok := cfg.Groups[name]
		if !ok {
			continue
		}
		connected := 0
		for _, p := range g.Profiles {
			if m, ok := states[p]; ok && m.Status == vpn.StatusConnected {
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
	// A saved choice (kongtrol config lang, dashboard toggle) wins; with none
	// the tray follows the OS language, same as the CLI.
	next := i18n.System()
	switch prefs.Language {
	case "es":
		next = i18n.ES
	case "en":
		next = i18n.EN
	}
	changed := next != trayLang
	trayLang = next
	trayPrefsMod = info.ModTime()
	return changed
}

func applyTrayMenuLanguage(labels trayMenuLabels) {
	systray.SetTooltip(trayT("tray.tooltip"))
	set := func(item *systray.MenuItem, title, tip string) {
		if item != nil {
			item.SetTitle(trayT(title))
			item.SetTooltip(trayT(tip))
		}
	}
	set(labels.dashboard, "tray.open_dashboard", "tray.open_dashboard_tip")
	set(labels.groups, "tray.groups", "tray.groups_tip")
	set(labels.tunnels, "tray.tunnels", "tray.tunnels_tip")
	set(labels.stopAll, "tray.stop_all", "tray.stop_all_tip")
	set(labels.quit, "tray.quit", "tray.quit_tip")
}

// sortedKeys returns m's keys in order, so the menu is stable across runs.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

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
