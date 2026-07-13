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
	"os"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/systray"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	"github.com/vpn-kongtrol/kongtrol/assets"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
)

var (
	cfg      *config.Config
	adapters map[string]vpn.VPNAdapter
	col      *monitor.Collector
	srv      *api.Server
	ks       security.KillSwitch
	leak     *security.LeakTester
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("VPN Kongtrol")
	systray.SetTooltip("VPN Kongtrol — multi-VPN orchestrator")
	setIcon(false)

	if err := initDaemon(); err != nil {
		systray.SetTitle("Kongtrol — config error")
		fmt.Fprintf(os.Stderr, "kongtrol-tray: init: %v\n", err)
	}

	// ── Menu items ────────────────────────────────────────────────────────

	mDashboard := systray.AddMenuItem("Open Dashboard", "Open browser at localhost:9741")
	systray.AddSeparator()

	// Per-profile connect/disconnect items.
	profileItems := make(map[string]*systray.MenuItem)
	for name := range cfg.VPNs {
		item := systray.AddMenuItem(fmt.Sprintf("  %s", name), "")
		profileItems[name] = item
	}

	systray.AddSeparator()
	mStatus := systray.AddMenuItem("Status: loading…", "")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Stop Kongtrol and exit")

	// ── Background: status refresh ────────────────────────────────────────

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateTrayStatus(mStatus, profileItems)
		}
	}()

	// ── Event loop ────────────────────────────────────────────────────────

	go func() {
		for {
			select {
			case <-mDashboard.ClickedCh:
				openBrowser(srv.Addr())

			case <-mQuit.ClickedCh:
				systray.Quit()

			default:
				// Check profile clicks.
				for name, item := range profileItems {
					select {
					case <-item.ClickedCh:
						go toggleProfile(name)
					default:
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func onExit() {
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	if col != nil {
		col.Stop()
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

	if cfg.Security.LeakDetection.Enabled {
		leak = security.NewLeakTester(60*time.Second, cfg.Security.IntegrityCheck.ExpectedIPs)
	}

	eng, _ := policy.New(cfg)

	col = monitor.NewCollector(adapters)
	col.Start(5 * time.Second)

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
		nil, // PolicyResolver is managed by the CLI daemon, not the tray
		nil, // DNSManager is managed by the CLI daemon, not the tray
		cfg.Security.DNSGuard.Enabled,
	)
	return srv.Start()
}

func updateTrayStatus(mStatus *systray.MenuItem, profileItems map[string]*systray.MenuItem) {
	snapshot := col.Snapshot()
	connectedCount := 0
	for _, m := range snapshot {
		if m.Status == vpn.StatusConnected {
			connectedCount++
		}
	}

	label := fmt.Sprintf("Status: %d tunnel(s) active", connectedCount)
	mStatus.SetTitle(label)
	setIcon(connectedCount > 0)

	for name, item := range profileItems {
		if m, ok := snapshot[name]; ok {
			status := string(m.Status)
			item.SetTitle(fmt.Sprintf("  %s [%s]", name, status))
		}
	}
}

func toggleProfile(name string) {
	adapter, ok := adapters[name]
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if adapter.Status() == vpn.StatusConnected {
		_ = adapter.Disconnect(ctx)
	} else {
		// TODO: inject credentials from keychain
		_ = adapter.Reconnect(ctx)
	}
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
