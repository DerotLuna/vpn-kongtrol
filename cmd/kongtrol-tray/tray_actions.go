package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// errNoConfig marks the first run: no kongtrol.yaml exists yet.
var errNoConfig = errors.New("no config file found")

// configExists reports whether any default config file exists.
func configExists() bool {
	for _, candidate := range config.DefaultPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

// initClient loads the config (for the API address and the menu's
// profiles/groups) with the machine-local dashboard bind/port preference
// applied, the local API token, and locates the kongtrol CLI the tray
// spawns the daemon with. With no config at all it still sets up the
// client (default address + preferences), so the first-run menu can start
// the daemon in setup mode and open the dashboard, and returns errNoConfig.
func initClient() error {
	var err error
	kongtrolExe, err = daemonctl.FindKongtrol(trayLang)
	if err != nil {
		return err
	}
	firstRun := !configExists()
	if firstRun {
		cfg = config.NewEmpty()
	} else if cfg, err = config.Load(""); err != nil {
		return err
	}
	config.ApplyDashboardPreferences(cfg)
	token, err := config.LoadOrCreateAPIToken()
	if err != nil {
		return err
	}
	client = daemonctl.New(daemonctl.APIBase(cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port), token)
	if firstRun {
		return errNoConfig
	}
	return nil
}

// ensureMu serializes ensureDaemon so concurrent clicks don't each wait on
// their own spawn; the shared spawn lock already prevents two daemons.
var ensureMu sync.Mutex

// startingDaemon is true while ensureDaemon runs, so the menu says
// "Iniciando Kongtrol…" instead of "detenido" in the meantime.
var startingDaemon atomic.Bool

// ensureDaemon makes sure the daemon answers, spawning it detached exactly
// like `kongtrol up` does when it doesn't.
func ensureDaemon() error {
	ensureMu.Lock()
	defer ensureMu.Unlock()
	startingDaemon.Store(true)
	defer startingDaemon.Store(false)
	sp := daemonctl.Spawner{
		Probe: client.Probe,
		Spawn: func(logPath string) (<-chan error, error) {
			return daemonctl.Spawn(kongtrolExe, []string{"daemon", "--plain"}, logPath, trayLang)
		},
		Lang: trayLang,
	}
	started, err := sp.Ensure(context.Background())
	if err != nil {
		var se *daemonctl.StartError
		if errors.As(err, &se) {
			for _, line := range se.LogTail {
				log.Printf("kongtrol-tray: daemon.log: %s", line)
			}
		}
		return err
	}
	if started {
		log.Printf("kongtrol-tray: started kongtrol daemon (%s)", client.Base)
	}
	return nil
}

// tunnelStatus returns the daemon's current status for profile name.
func tunnelStatus(name string) (vpn.Status, error) {
	statuses, err := client.Statuses()
	if err != nil {
		return "", err
	}
	return statuses[name], nil
}

// toggleProfile disconnects a connected profile, or asks the daemon to
// connect it (asynchronously; a failure shows up as the tunnel's last
// error on the next status refresh).
func toggleProfile(name string) {
	if err := ensureDaemon(); err != nil {
		notifyError(trayT("tray.err_start"), err)
		return
	}
	st, err := tunnelStatus(name)
	if err != nil {
		notifyError(trayT("tray.err_unreachable"), err)
		return
	}
	if st == vpn.StatusConnected {
		disconnectProfile(name)
	} else {
		connectProfile(name)
	}
}

// toggleGroup connects every disconnected profile in the group, unless every
// profile in it is already connected, in which case it disconnects all of
// them — a single click, since the tray menu has no separate
// connect/disconnect buttons per group.
func toggleGroup(name string) {
	g, ok := cfg.Groups[name]
	if !ok {
		return
	}
	if err := ensureDaemon(); err != nil {
		notifyError(trayT("tray.err_start"), err)
		return
	}
	statuses, err := client.Statuses()
	if err != nil {
		notifyError(trayT("tray.err_unreachable"), err)
		return
	}
	allConnected := true
	for _, p := range g.Profiles {
		if statuses[p] != vpn.StatusConnected {
			allConnected = false
			break
		}
	}
	for _, p := range g.Profiles {
		if allConnected {
			go disconnectProfile(p)
		} else if statuses[p] != vpn.StatusConnected {
			go connectProfile(p)
		}
	}
}

func connectProfile(name string) {
	if _, err := client.Connect(name); err != nil {
		notifyError(trayF("tray.err_connect", name), err)
	}
}

func disconnectProfile(name string) {
	// A provider app without a CLI answers ErrManualDisconnect: not an
	// error, the menu item shows "Disconnect in <App>" until it drops.
	if err := client.Disconnect(name); err != nil && !errors.Is(err, vpn.ErrManualDisconnect) {
		notifyError(trayF("tray.err_disconnect", name), err)
	}
}

// openDashboard starts the daemon if needed and opens its page.
func openDashboard() { openDashboardPage("/") }

// setupPagePath is the dashboard page a first run opens: Perfiles, which
// shows the "create your first VPN" guide while the daemon runs in setup
// mode (no kongtrol.yaml yet). No page
// fragment of its own; the only fragment is PanelURL's one-time "#k="
// sign-in nonce (ADR-0039). If a Windows URL handler drops it, the page
// shows how to open it from Kongtrol instead (no verificado on every
// Windows version).
const setupPagePath = "/profiles"

// openDashboardPage starts the daemon if needed and opens path on it. It
// returns false when either step failed (the user was already notified).
func openDashboardPage(path string) bool {
	if err := ensureDaemon(); err != nil {
		notifyError(trayT("tray.err_start"), err)
		return false
	}
	// PanelURL adds a one-time sign-in nonce; on failure it is the plain
	// URL and the page explains how to open it from Kongtrol.
	panel, _ := client.PanelURL(path)
	if err := openBrowser(panel); err != nil {
		notifyError(trayF("tray.err_browser", client.Base+path), err)
		return false
	}
	return true
}

// stopKongtrol is the tray's "quit and disconnect everything": it asks the
// daemon to disconnect all tunnels and stop, then closes the tray. If the
// shutdown request fails the tray stays open so the user sees it failed.
func stopKongtrol() {
	if client.Probe() {
		if err := client.Shutdown(); err != nil {
			notifyError(trayT("tray.err_shutdown"), err)
			return
		}
		log.Println("kongtrol-tray: asked the kongtrol daemon to disconnect everything and stop")
	}
	systray.Quit()
}

var (
	// helperItem is the "Install helper (once)" menu item; nil until the
	// full menu is built.
	helperItem atomic.Pointer[systray.MenuItem]
	// helperInstalling is true while an install waits on the OS prompt.
	helperInstalling atomic.Bool
)

// refreshHelperItem shows the install item only while the privileged helper
// is missing (helperNeedsInstall), on every OS and also in the first-run
// menu, and tells the user about it once (tray.helper_first_run).
func refreshHelperItem() {
	item := helperItem.Load()
	if item == nil {
		return
	}
	if helperInstalling.Load() {
		item.SetTitle(trayT("tray.helper_installing"))
		item.Disable()
		item.Show()
		return
	}
	need := helperState()
	if need == helperOK {
		item.Hide()
		return
	}
	title, tip := helperItemText(need)
	item.SetTitle(title)
	item.SetTooltip(tip)
	item.Enable()
	item.Show()
	if need == helperMissing && claimOnce(helperNoticeMarker()) {
		msg := trayT("tray.helper_first_run")
		notifyError(msg, nil)
		desktopNotify("Kongtrol", msg)
	}
}

// Probes behind helperNeedsInstall; tests replace them.
var (
	daemonHelperStatus = func() (daemonctl.HelperStatus, error) { return client.HelperStatus() }
	helperIsElevated   = privops.IsElevated
	pingHelper         = func(ctx context.Context) (privops.PingResult, error) {
		return privops.NewClient().Ping(ctx)
	}
)

// helperNeed is what the helper item offers.
type helperNeed int

const (
	helperOK       helperNeed = iota // nothing to do: item hidden
	helperMissing                    // "Instalar ayudante (una vez)"
	helperOutdated                   // "Actualizar ayudante (una vez)"
)

// helperState asks the daemon first. When no daemon answers (fresh
// install, first-run menu, daemon stopped) it asks the helper itself, with
// the daemon's rule: not elevated and no helper answering with our
// protocol; one that answers with an older protocol is outdated.
func helperState() helperNeed {
	if st, err := daemonHelperStatus(); err == nil {
		switch {
		case !st.NeedsInstall:
			return helperOK
		case st.Outdated:
			return helperOutdated
		}
		return helperMissing
	}
	if helperIsElevated() {
		return helperOK
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := pingHelper(ctx)
	switch {
	case errors.Is(err, privops.ErrHelperOutdated):
		// The client reports a reply with another protocol as this error.
		return helperOutdated
	case err != nil:
		return helperMissing
	case res.Protocol != privops.ProtocolVersion:
		return helperOutdated
	}
	return helperOK
}

// helperItemText is the helper item's title and tooltip for need.
func helperItemText(need helperNeed) (title, tip string) {
	if need == helperOutdated {
		return trayT("tray.helper_update"), trayT("tray.helper_update_tip")
	}
	return trayT("tray.helper_install"), trayT("tray.helper_install_tip")
}

// helperNoticeMarker records that the "helper missing" notice was shown,
// so it appears once, not at every login (the menu item stays).
func helperNoticeMarker() string {
	return filepath.Join(daemonctl.StateDir(), "run", "helper-notice-shown")
}

// claimOnce creates path and reports whether this call created it. Without
// a writable state dir it reports false (nothing shown on its own).
func claimOnce(path string) bool {
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

// addHelperMenuItem adds the hidden "Install helper (once)" item; shown by
// refreshHelperItem.
func addHelperMenuItem() *systray.MenuItem {
	mHelper := systray.AddMenuItem(trayT("tray.helper_install"), trayT("tray.helper_install_tip"))
	mHelper.Hide()
	helperItem.Store(mHelper)
	onClick(mHelper, installHelper)
	return mHelper
}

// installHelper asks the daemon to install (or update) the privileged
// helper: the OS shows one permission prompt (UAC / macOS / polkit
// dialog). While it runs the item, status line and icon tooltip say so;
// afterwards a desktop notification confirms it ("ya no hace falta volver
// a instalarlo") and the item disappears at once (status re-checked right
// away, not on the next poll), or the status line says what to do.
func installHelper() {
	if !helperInstalling.CompareAndSwap(false, true) {
		return
	}
	started := true
	err := func() error {
		defer helperInstalling.Store(false)
		if err := ensureDaemon(); err != nil {
			started = false
			return err
		}
		refreshHelperItem()
		notifyError(trayT("tray.helper_installing"), nil)
		_, err := client.InstallHelper(trayLang)
		return err
	}()
	var he *daemonctl.HelperInstallError
	switch {
	case !started:
		notifyError(trayT("tray.err_start"), err)
	case errors.As(err, &he):
		notifyError(he.Message+" · "+he.Hint, err)
	case err != nil:
		notifyError(trayT("tray.helper_install_failed"), err)
	default:
		msg := trayT("tray.helper_installed")
		notifyError(msg, nil)
		desktopNotify("Kongtrol", msg)
	}
	refreshHelperItem()
	requestRefresh()
}
