// Command kongtrol-tray is the system tray frontend for Kongtrol.
// It is a client of the kongtrol daemon (`kongtrol daemon`), exactly like
// `kongtrol up`: on start it probes the daemon's local API and, when none
// answers, spawns a detached daemon with the kongtrol CLI found next to the
// tray executable (or in PATH). Every menu action goes through the API.
// Quitting the tray never stops the daemon or its tunnels; only the
// explicit "stop kongtrol" item (or `kongtrol down`) does.
//
// Build note: requires CGO on Linux (GTK) and macOS (Cocoa). Cross-
// compilation requires a native build per target OS — use goreleaser.
package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

var (
	cfg          *config.Config
	client       *daemonctl.Client
	kongtrolExe  string
	trayLang     = i18n.System() // until preferences.json says otherwise
	trayPrefsMod time.Time
)

func trayT(key string) string { return i18n.T(trayLang, key) }

func trayF(key string, args ...any) string { return i18n.F(trayLang, key, args...) }

func main() {
	setupLogging()
	if !acquireSingleInstance() {
		log.Println("kongtrol-tray: another instance is already running, exiting")
		return
	}
	systray.Run(onReady, onExit)
}

// setupLogging redirects the standard `log` package (used both by our own
// code and internally by fyne.io/systray for diagnostics like icon-set
// failures) to a file. The tray is built with -H=windowsgui (see Makefile)
// so it has no attached console — without this, that output would just be
// lost. Falls back to stderr if the log file can't be opened.
func setupLogging() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".kongtrol")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "tray.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.Printf("kongtrol-tray starting (pid %d)", os.Getpid())
}

// onExit runs when the tray quits. It deliberately leaves the daemon and
// its tunnels running: the tray is only a viewer.
func onExit() {
	log.Println("kongtrol-tray: exiting; the kongtrol daemon keeps running")
}
