package main

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
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

// ensureDaemon makes sure the daemon answers, spawning it detached exactly
// like `kongtrol up` does when it doesn't.
func ensureDaemon() error {
	ensureMu.Lock()
	defer ensureMu.Unlock()
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

// setupPagePath is the dashboard page a first run opens: the Studio, which
// switches to its VPN profile tab and shows the "create your first VPN"
// guide while the daemon runs in setup mode (no kongtrol.yaml yet). No URL
// fragment: Windows' url.dll handler is not documented to keep one.
const setupPagePath = "/studio.html"

// openDashboardPage starts the daemon if needed and opens path on it. It
// returns false when either step failed (the user was already notified).
func openDashboardPage(path string) bool {
	if err := ensureDaemon(); err != nil {
		notifyError(trayT("tray.err_start"), err)
		return false
	}
	if err := openBrowser(client.Base + path); err != nil {
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

// refreshHelperItem shows the install item only while the daemon says the
// privileged helper is missing. An older daemon without the endpoint
// hides it.
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
	st, err := client.HelperStatus()
	if err != nil || !st.NeedsInstall {
		item.Hide()
		return
	}
	item.SetTitle(trayT("tray.helper_install"))
	item.SetTooltip(trayT("tray.helper_install_tip"))
	item.Enable()
	item.Show()
}

// installHelper asks the daemon to install the privileged helper: the OS
// shows one permission prompt (UAC / macOS / polkit dialog), then the
// status line says whether it worked and, if not, what to do.
func installHelper() {
	if !helperInstalling.CompareAndSwap(false, true) {
		return
	}
	err := func() error {
		defer helperInstalling.Store(false)
		if err := ensureDaemon(); err != nil {
			notifyError(trayT("tray.err_start"), err)
			return nil
		}
		refreshHelperItem()
		notifyError(trayT("tray.helper_installing"), nil)
		_, err := client.InstallHelper(trayLang)
		return err
	}()
	var he *daemonctl.HelperInstallError
	switch {
	case errors.As(err, &he):
		notifyError(he.Message+" · "+he.Hint, err)
	case err != nil:
		notifyError(trayT("tray.helper_install_failed"), err)
	default:
		notifyError(trayT("tray.helper_installed"), nil)
	}
	refreshHelperItem()
}
