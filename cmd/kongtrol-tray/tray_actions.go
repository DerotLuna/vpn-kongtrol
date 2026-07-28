package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"go.uber.org/zap"
)

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
