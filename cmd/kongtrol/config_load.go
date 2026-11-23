package main

import (
	"fmt"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"go.uber.org/zap"
)

func loadConfig() error {
	var err error
	setupMode = false
	activeCfgPath, err = resolveConfigPath(cfgPath)
	switch {
	case err != nil && allowSetupMode && cfgPath == "":
		// First run of the daemon (spawned by the tray before any
		// kongtrol.yaml exists): serve the dashboard with no profiles so
		// the user can create the first one there. The file is written to
		// the first default path once that profile is valid.
		setupMode = true
		activeCfgPath = config.DefaultPaths()[0]
		cfg = config.NewEmpty()
		config.ApplyDashboardPreferences(cfg)
	case err != nil:
		return err
	default:
		cfg, err = config.Load(activeCfgPath)
		if err != nil {
			return localizeConfigErr(err)
		}
		config.ApplyDashboardPreferences(cfg)
		if err := config.Validate(cfg); err != nil {
			return localizeConfigErr(err)
		}
	}
	apiToken, err = config.LoadOrCreateAPIToken()
	if err != nil {
		return err
	}

	// Instantiate adapters from config.
	initial := make(map[string]vpn.VPNAdapter, len(cfg.VPNs))
	for name, vpnCfg := range cfg.VPNs {
		a, err := app.NewProfileAdapter(vpnCfg)
		if err != nil {
			return fmt.Errorf("config: profile %q: %w", name, err)
		}
		initial[name] = a
	}
	adapters = vpn.NewSet(initial)

	// Route manager (OS-specific).
	routeMgr = routing.NewRouteManager()

	// Policy engine.
	engine, err = policy.New(cfg)
	if err != nil {
		return err
	}

	// Security.
	ks = security.NewKillSwitch() // legacy all-traffic rules: only removed
	if destBlocker == nil {
		destBlocker = security.NewDestinationBlocker()
	}
	killSwitchSvc = app.NewKillSwitchService(cfg, adapters, ks)
	killSwitchSvc.SetBlocker(destBlocker)
	killSwitchSvc.SetRouteLister(routeMgr.List)
	killSwitchSvc.SetOnBlock(onKillSwitchBlock)
	// Public IP lookup (and the leak check built on it): always present so
	// it can be toggled at runtime, but it makes no request unless the user
	// opted in (security.public_ip_check). Kill switch and DNS guard never
	// turn it on.
	leak = newPublicIPChecker(cfg)

	// Monitor.
	col = monitor.NewCollector(adapters)
	_ = col.LoadHistory(metricsHistoryPath())
	col.Start(5 * time.Second)

	// Watchdog — auto-reconnect on unexpected disconnects.
	log, _ := zap.NewProduction()
	watchdog = monitor.NewWatchdog(adapters, connectProfile, log)
	watchdog.ConfigureHealthCheck(
		parseDuration(cfg.Monitor.HealthCheck.Interval, 30*time.Second),
		parseDuration(cfg.Monitor.HealthCheck.Timeout, 10*time.Second),
		healthCheckProfile,
	)
	watchdog.ConfigureFailover(profileFailover(cfg))
	adapterSync = app.NewAdapterSync(adapters, cfg, watchdog.Forget)
	watchdog.SetEventCallback(onWatchdogEvent)

	// DNS manager — reference-counted guard across simultaneous tunnels.
	dnsGuard := security.NewDNSGuard()
	dnsMgr = monitor.NewDNSManager(dnsGuard, log)

	// Policy resolver — background DNS re-resolution for domain-based split tunnel.
	policyResolver = monitor.NewPolicyResolver(cfg, routeMgr, log)
	policyResolver.SetTunnelProbe(probeTunnel)
	killSwitchSvc.SetDestinations(policyResolver)
	policyWatch = &monitor.PolicyWatch{
		Matcher:         policyResolver,
		Status:          profileStatus,
		Connect:         connectProfile,
		ConnectConflict: policyAutoConnectConflict,
		Events:          daemonEvents,
		OnEvent:         logPolicyEvent,
	}
	// Recent destinations: memory only, never on disk or sent (ADR-0054).
	recentConns = monitor.NewRecentConnections(cfg.RecentConnectionsEnabled())
	recentConns.Host = policyResolver.HostOf
	recentConns.Via = tunnelOfLocalAddr
	scheduler = monitor.NewScheduler(
		cfg,
		parseDuration(cfg.Monitor.Scheduler.Interval, time.Minute),
		connectProfile,
		disconnectProfile,
		func(profile string) vpn.Status {
			if a, ok := adapters.Get(profile); ok {
				return a.Status().Normalize()
			}
			return vpn.StatusDisconnected
		},
		log,
	)
	splitDNSMgr = monitor.NewSplitDNSManager(
		policyResolver,
		parseDuration(cfg.Monitor.SplitDNS.Interval, 60*time.Second),
		log,
	)
	profileSvc.Store(buildProfileService())

	applyAuditConfig(cfg)

	return nil
}
