// Command kongtrol-tray is the system tray frontend for VPN Kongtrol.
// It starts the full Kongtrol daemon internally and exposes tunnel
// status and quick actions from the OS system tray.
//
// Build note: requires CGO (GTK on Linux, Cocoa on macOS, Win32 on Windows).
// Cross-compilation requires a native build per target OS — use goreleaser.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/assets"
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

func initDaemon() error {
	var err error
	cfg, err = config.Load("")
	if err != nil {
		return err
	}

	adapters = make(map[string]vpn.VPNAdapter)
	for name, vpnCfg := range cfg.VPNs {
		a, err := vpn.New(vpnCfg.Type)
		if err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		adapters[name] = a
	}

	routeMgr := routing.NewRouteManager()
	ks = security.NewKillSwitch()

	if cfg.Security.LeakDetection.Enabled ||
		cfg.Security.KillSwitch.Enabled ||
		cfg.Security.DNSGuard.Enabled {
		leak = security.NewLeakTester(60*time.Second, cfg.Security.IntegrityCheck.ExpectedIPs)
	}

	eng, _ := policy.New(cfg)

	col = monitor.NewCollector(adapters)
	col.Start(5 * time.Second)

	zapLog, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	watchdog = monitor.NewWatchdog(adapters, func(ctx context.Context, name string) error {
		return trayConnect(ctx, name)
	}, zapLog)
	dnsMgr = monitor.NewDNSManager(security.NewDNSGuard(), zapLog)
	resolver = monitor.NewPolicyResolver(cfg, routeMgr, zapLog)
	killSvc = app.NewKillSwitchService(cfg, adapters, ks)
	buildProfileService := func() {
		profileSvc.Store(app.NewProfileService(app.ProfileServiceDeps{
			Cfg:            cfg,
			Adapters:       adapters,
			Watchdog:       watchdog,
			PolicyResolver: resolver,
			DNSManager:     dnsMgr,
			ApplyKill:      killSvc.Apply,
			EmitWarn:       func(message string) { log.Println(message) },
			EmitStderr:     func(message string) { log.Println(message) },
			LogAudit: func(level, action, profile, message string) {
				log.Printf("%s %s profile=%s %s", level, action, profile, message)
			},
		}))
	}
	buildProfileService()
	daemonCtx, cancel := context.WithCancel(context.Background())
	daemonCancel = cancel
	watchdog.Start(daemonCtx)
	resolver.Start(daemonCtx)
	if leak != nil {
		leak.Start(daemonCtx, func(result security.LeakResult) {
			if result.HasLeak {
				log.Printf("kongtrol-tray: leak detected: %s", result.Reason)
			}
		})
	}
	apiToken, err := config.LoadOrCreateAPIToken()
	if err != nil {
		return err
	}

	srv = api.NewServer(
		cfg.Monitor.Dashboard.Bind,
		cfg.Monitor.Dashboard.Port,
		adapters,
		col,
		routeMgr,
		ks,
		cfg.Security.KillSwitch.Enabled,
		leak,
		eng,
		resolver,
		"", // config path not used by tray policy editor
		func(newCfg *config.Config, _ *policy.Engine) {
			cfg = newCfg
			killSvc = app.NewKillSwitchService(cfg, adapters, ks)
			buildProfileService()
		},
		func(*config.Config) {
			_ = killSvc.Apply()
			applyTrayDNSGuardState()
		},
		dnsMgr,
		cfg.Security.DNSGuard.Enabled,
		trayConnect,
		trayDisconnect,
		apiToken,
		nil, // graceful /api/v1/shutdown not wired into the tray's systray.Quit() lifecycle
	)
	return srv.Start()
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
func toggleProfile(name string) {
	adapter, ok := adapters[name]
	if !ok {
		return
	}
	if adapter.Status().Normalize() == vpn.StatusConnected {
		disconnectProfile(name)
	} else {
		connectProfile(name)
	}
}

// toggleGroup connects every disconnected profile in the group, unless every
// profile in it is already connected, in which case it disconnects all of
// them — mirroring the dashboard's per-group connect/disconnect behavior
// (internal/api/handlers.go handleConnectGroup/handleDisconnectGroup) but as
// a single click since the tray menu has no separate connect/disconnect
// buttons per group.
func toggleGroup(name string) {
	g, ok := cfg.Groups[name]
	if !ok {
		return
	}
	allConnected := true
	for _, p := range g.Profiles {
		if a, ok := adapters[p]; !ok || a.Status().Normalize() != vpn.StatusConnected {
			allConnected = false
			break
		}
	}
	for _, p := range g.Profiles {
		if allConnected {
			go disconnectProfile(p)
		} else {
			go connectProfile(p)
		}
	}
}

func connectProfile(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := trayConnect(ctx, name); err != nil {
		log.Printf("kongtrol-tray: connect %q: %v", name, err)
	}
}

func disconnectProfile(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := trayDisconnect(ctx, name); err != nil {
		log.Printf("kongtrol-tray: disconnect %q: %v", name, err)
	}
}

func trayConnect(ctx context.Context, name string) error {
	svc := profileSvc.Load()
	if svc == nil {
		return fmt.Errorf("profile service unavailable")
	}
	return svc.Connect(ctx, name)
}

func trayDisconnect(ctx context.Context, name string) error {
	svc := profileSvc.Load()
	if svc == nil {
		return fmt.Errorf("profile service unavailable")
	}
	return svc.Disconnect(ctx, name)
}

// setIcon updates the system tray icon based on connection state.
// Connected: full logo. Disconnected: dimmed (same icon — OS handles dimming).
// Icon is resized to 32×32 for systray compatibility.
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

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
