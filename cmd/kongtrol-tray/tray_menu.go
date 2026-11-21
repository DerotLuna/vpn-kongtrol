package main

import (
	"errors"
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
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

func onReady() {
	refreshTrayLanguage()
	systray.SetTitle("Kongtrol")
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
	if err := applyTrayIcon(assets.BadgeNone, trayIconTheme()); err != nil {
		log.Printf("kongtrol-tray: tray icon: %v", err)
	}
	setupMenu()
}

// refreshCh asks the status loop for an immediate refresh (after a click
// that changes state), instead of waiting for the next 3 s tick.
var refreshCh = make(chan struct{}, 1)

func requestRefresh() {
	select {
	case refreshCh <- struct{}{}:
	default:
	}
}

// setupMenu builds the full menu, or the first-run / error menu when the
// tray cannot work yet. It runs again (after systray.ResetMenu) once a
// first-run user has created the config.
//
// Layout (top to bottom): status header (VPNs up, kill switch, public IP)
// and the lines that ask for an action (start Kongtrol, helper, policy);
// primary actions (panel, VPN connections, groups); the version; the two
// exit items.
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
	// The installer's "create my first VPN" checkbox with profiles already
	// there (data kept from an earlier install): open the page anyway.
	if p := takeOpenRequest(); p != "" {
		go func() {
			if !openDashboardPage(p) {
				log.Printf("kongtrol-tray: --open %s failed", p)
			}
		}()
	}

	m := &trayMenu{}

	// ── Status header ─────────────────────────────────────────────────────
	m.header = systray.AddMenuItem(trayT("tray.status_loading"), "")
	m.header.Disable()
	noticeMu.Lock()
	statusItem = m.header
	noticeMu.Unlock()
	addKillSwitchMenuItem()
	addPublicIPMenuItem()
	// Lines that ask for one action; hidden unless needed.
	m.start = systray.AddMenuItem(trayT("tray.start"), trayT("tray.start_tip"))
	m.start.Hide()
	onClick(m.start, func() { startKongtrol(m) })
	// Shown only while the privileged helper is missing or outdated.
	addHelperMenuItem()
	// "Conectar <perfil>" after a policy notice (see tray_policy.go).
	addPolicyMenuItem()
	systray.AddSeparator()

	// ── Primary actions ───────────────────────────────────────────────────
	m.dash = systray.AddMenuItem(trayT("tray.open_dashboard"), trayT("tray.open_dashboard_tip"))
	onClick(m.dash, openDashboard)
	diagnostics := systray.AddMenuItem(trayT("tray.open_diagnostics"), trayT("tray.open_diagnostics"))
	onClick(diagnostics, openDiagnostics)

	// One slot per VPN profile; renderTunnels relabels them so connected
	// VPNs come first.
	m.tunnels = systray.AddMenuItem(trayT("tray.tunnels"), trayT("tray.tunnels_tip"))
	for _, name := range sortedKeys(cfg.VPNs) {
		s := &slotItem{item: m.tunnels.AddSubMenuItemCheckbox(name, "", false), name: name}
		m.profiles = append(m.profiles, s)
		onClick(s.item, func() {
			if n := m.slotName(s); n != "" {
				toggleProfile(n)
				requestRefresh()
			}
		})
	}

	// Groups — clicking connects every disconnected profile in the group,
	// or disconnects all of them when every one is already connected.
	if len(cfg.Groups) > 0 {
		m.groups = systray.AddMenuItem(trayT("tray.groups"), trayT("tray.groups_tip"))
		for _, name := range sortedKeys(cfg.Groups) {
			s := &slotItem{item: m.groups.AddSubMenuItemCheckbox(name, "", false), name: name}
			m.grpSlots = append(m.grpSlots, s)
			onClick(s.item, func() {
				toggleGroup(name)
				requestRefresh()
			})
		}
	}
	systray.AddSeparator()

	// ── Version ───────────────────────────────────────────────────────────
	m.ver = systray.AddMenuItem("", "")
	m.ver.Disable()
	m.refreshVersion(false)
	systray.AddSeparator()

	// ── Exit ──────────────────────────────────────────────────────────────
	m.stopAll = systray.AddMenuItem(trayT("tray.stop_all"), trayT("tray.stop_all_tip"))
	onClick(m.stopAll, stopKongtrol)
	m.quit = systray.AddMenuItem(trayT("tray.quit"), trayT("tray.quit_tip"))
	// Quitting closes only the tray; the daemon keeps its tunnels up.
	onClick(m.quit, systray.Quit)
	// Left click: branded flyout where supported (flyout*.go, ADR-0036);
	// the menu above stays on the right click and as the fallback.
	setupFlyout()

	// ── Background: daemon start + status refresh ─────────────────────────

	go func() {
		if err := ensureDaemon(); err != nil {
			notifyError(trayT("tray.err_start"), err)
		}
		updateTrayStatus(m)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
			case <-refreshCh:
			}
			updateTrayStatus(m)
		}
	}()
}

// startKongtrol is "Iniciar Kongtrol": the menu shows it only while the
// daemon is stopped (after `kongtrol down`). It starts the daemon exactly
// like the tray does at login; the VPNs stay down until the user connects.
func startKongtrol(m *trayMenu) {
	m.start.Disable()
	m.header.SetTitle(trayT("tray.header.starting"))
	if err := ensureDaemon(); err != nil {
		notifyError(trayT("tray.err_start"), err)
	}
	requestRefresh()
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
func claimFirstRunOpen() bool { return claimOnce(firstRunMarker()) }

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
	// Automatic opening happens once (marker); an explicit --open from the
	// installer's checkbox always opens (ADR-0050). The marker is claimed
	// either way, so the next start does not open it again by itself.
	claimed := claimFirstRunOpen()
	if asked := takeOpenRequest() != ""; claimed || asked {
		go openSetup()
	}
	// A fresh install also needs the helper (macOS/Linux have no installer
	// step for it): offer it here too, not only in the full menu.
	addHelperMenuItem()
	go refreshHelperItem()
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
				refreshHelperItem()
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

// updateTrayStatus polls the daemon and renders the whole menu. It keeps
// running while the daemon is stopped, so the menu recovers by itself when
// the daemon comes back (CLI `kongtrol up`, the dashboard, "Iniciar").
func updateTrayStatus(m *trayMenu) {
	if refreshTrayLanguage() {
		applyTrayMenuLanguage(m)
	}

	notice := activeNotice()
	states, err := client.TunnelStates()
	d := classifyDaemon(err, startingDaemon.Load())
	names := sortedKeys(cfg.VPNs)
	if err != nil {
		states = map[string]monitor.TunnelMetrics{}
	}
	views := trayTunnelViews(names, states)
	header, badge := trayHeader(d, views)
	m.renderDaemon(d, header, badge)
	m.renderTunnels(views)
	m.renderGroups(states)
	m.refreshVersion(d == daemonRunning)
	if err != nil {
		tip := trayT("tray.daemon_unreachable")
		if d == daemonStopped {
			tip = trayT("tray.tooltip.stopped")
		}
		if notice != "" {
			m.header.SetTitle(notice)
			tip = notice
		}
		systray.SetTooltip(tip)
		hidePolicyAndStatusLines()
		// No daemon: refreshHelperItem asks the helper directly.
		refreshHelperItem()
		return
	}
	// The daemon answered: the flyout may skip its own probe and is
	// pre-warmed the first time (flyout.go, ADR-0074).
	noteDaemonAnswered()
	refreshHelperItem()
	pollPolicyEvents(states)
	refreshKillSwitchItem()
	refreshPublicIPItem()
	notice = activeNotice() // a new policy notice shows at once
	if notice != "" {
		m.header.SetTitle(notice)
	}

	// A provider waiting for the user in its app (or dropped) is the most
	// useful thing the icon tooltip can say.
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	switch tip := trayWaitingTooltip(states, sorted); {
	case notice != "":
		systray.SetTooltip(notice)
	case tip != "":
		systray.SetTooltip(tip)
	default:
		systray.SetTooltip(header + " · " + trayT("tray.tooltip"))
	}
}

// hidePolicyAndStatusLines hides the kill switch and public IP lines while
// no daemon answers: their last values would be stale.
func hidePolicyAndStatusLines() {
	policyMu.Lock()
	ks := ksItem
	policyMu.Unlock()
	if ks != nil {
		ks.Hide()
	}
	publicIPMu.Lock()
	ip := publicIPItem
	publicIPMu.Unlock()
	if ip != nil {
		ip.Hide()
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

func applyTrayMenuLanguage(m *trayMenu) {
	systray.SetTooltip(trayT("tray.tooltip"))
	set := func(item *systray.MenuItem, title, tip string) {
		if item != nil {
			item.SetTitle(trayT(title))
			item.SetTooltip(trayT(tip))
		}
	}
	set(m.dash, "tray.open_dashboard", "tray.open_dashboard_tip")
	set(m.groups, "tray.groups", "tray.groups_tip")
	set(m.tunnels, "tray.tunnels", "tray.tunnels_tip")
	set(m.start, "tray.start", "tray.start_tip")
	set(m.stopAll, "tray.stop_all", "tray.stop_all_tip")
	set(m.quit, "tray.quit", "tray.quit_tip")
	m.verText = ""
	// Profile / group titles carry translated state words: re-render all.
	for _, s := range append(append([]*slotItem(nil), m.profiles...), m.grpSlots...) {
		s.last = itemView{}
	}
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
