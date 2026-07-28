// Command kongtrol-tray is the system tray frontend for VPN Kongtrol.
// It starts the full Kongtrol daemon internally and exposes tunnel
// status and quick actions from the OS system tray.
//
// Build note: requires CGO (GTK on Linux, Cocoa on macOS, Win32 on Windows).
// Cross-compilation requires a native build per target OS — use goreleaser.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/ciscoanyconnect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/cloudflarewarp"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/globalprotect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/tailscale"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

var (
	cfg          *config.Config
	adapters     map[string]vpn.VPNAdapter
	col          *monitor.Collector
	srv          *api.Server
	ks           security.KillSwitch
	leak         *security.LeakTester
	profileSvc   atomic.Pointer[app.ProfileService]
	watchdog     *monitor.Watchdog
	dnsMgr       *monitor.DNSManager
	resolver     *monitor.PolicyResolver
	killSvc      *app.KillSwitchService
	daemonCancel context.CancelFunc
	trayLang     = i18n.ES
	trayPrefsMod time.Time
)

func trayT(key string) string { return i18n.T(trayLang, key) }

func trayF(key string, args ...any) string { return i18n.F(trayLang, key, args...) }

type trayMenuLabels struct {
	dashboard *systray.MenuItem
	groups    *systray.MenuItem
	tunnels   *systray.MenuItem
	quit      *systray.MenuItem
}

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
// lost, and the app would die whenever the terminal that happened to launch
// it was closed. Falls back to stderr if the log file can't be opened.
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

func onExit() {
	if daemonCancel != nil {
		daemonCancel()
	}
	if watchdog != nil {
		watchdog.Stop()
	}
	if resolver != nil {
		resolver.Stop()
	}
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	if col != nil {
		col.Stop()
	}
	if dnsMgr != nil {
		dnsMgr.ForceRestore()
	}
	if leak != nil {
		leak.Stop()
	}
	if ks != nil {
		_ = ks.Disable()
	}
}
