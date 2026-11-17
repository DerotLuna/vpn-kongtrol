package main

import (
	"fmt"
	"os"
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
	activeCfgPath, err = resolveConfigPath(cfgPath)
	if err != nil {
		return err
	}
	cfg, err = config.Load(activeCfgPath)
	if err != nil {
		return localizeConfigErr(err)
	}
	config.ApplyDashboardPreferences(cfg)
	if err := config.Validate(cfg); err != nil {
		return localizeConfigErr(err)
	}
	apiToken, err = config.LoadOrCreateAPIToken()
	if err != nil {
		return err
	}

	// Instantiate adapters from config.
	adapters = make(map[string]vpn.VPNAdapter)
	for name, vpnCfg := range cfg.VPNs {
		a, err := vpn.New(vpnCfg.Type)
		if err != nil {
			return fmt.Errorf("config: profile %q: %w", name, err)
		}
		// Pre-seed config so Status() can probe externally-started tunnels.
		if c, ok := a.(vpn.Configurable); ok {
			c.Configure(vpn.AdapterConfig{
				Host:       vpnCfg.Host,
				Port:       vpnCfg.Port,
				TunnelName: vpnCfg.TunnelName,
				CertPath:   vpnCfg.Auth.Cert,
				KeyPath:    vpnCfg.Auth.Key,
				ConfigPath: vpnCfg.ConfigFile,
				Username:   vpnCfg.Auth.Username,
				Extra:      vpnCfg.AdapterExtra(),
			})
		}
		adapters[name] = a
	}

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
	watchdog.ConfigureFailover(profileFailover())
	watchdog.SetEventCallback(onWatchdogEvent)

	// DNS manager — reference-counted guard across simultaneous tunnels.
	dnsGuard := security.NewDNSGuard()
	dnsMgr = monitor.NewDNSManager(dnsGuard, log)

	// Policy resolver — background DNS re-resolution for domain-based split tunnel.
	policyResolver = monitor.NewPolicyResolver(cfg, routeMgr, log)
	policyResolver.SetTunnelProbe(probeTunnel)
	killSwitchSvc.SetDestinations(policyResolver)
	policyWatch = &monitor.PolicyWatch{
		Matcher: policyResolver,
		Status:  profileStatus,
		Connect: connectProfile,
		Events:  daemonEvents,
		OnEvent: logPolicyEvent,
	}
	scheduler = monitor.NewScheduler(
		cfg,
		parseDuration(cfg.Monitor.Scheduler.Interval, time.Minute),
		connectProfile,
		disconnectProfile,
		func(profile string) vpn.Status {
			if a, ok := adapters[profile]; ok {
				return a.Status().Normalize()
			}
			return vpn.StatusDisconnected
		},
		log,
	)
	splitDNSMgr = monitor.NewSplitDNSManager(
		cfg,
		policyResolver,
		parseDuration(cfg.Monitor.SplitDNS.Interval, 60*time.Second),
		log,
	)
	profileSvc.Store(buildProfileService())

	if audit != nil {
		_ = audit.Close()
		audit = nil
	}
	if cfg.Security.AuditLog.Path != "" {
		var hmacKey []byte
		if cfg.Security.AuditLog.Sign {
			hmacKey = loadOrCreateAuditHMACKey()
		}
		a, err := security.NewAuditLogger(cfg.Security.AuditLog.Path, cfg.Security.AuditLog.Sign, hmacKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.audit.init_warn", err)))
		} else {
			audit = a
		}
	}

	return nil
}
