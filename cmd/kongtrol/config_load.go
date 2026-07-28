package main

import (
	"fmt"
	"os"
	"strconv"
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
		return err
	}
	applyDashboardPreferences(cfg)
	if err := config.Validate(cfg); err != nil {
		return err
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
				Extra: map[string]string{
					"protocol":            vpnCfg.Protocol,
					"allow_insecure_cert": strconv.FormatBool(vpnCfg.AllowInsecureCert),
				},
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
	ks = security.NewKillSwitch()
	killSwitchSvc = app.NewKillSwitchService(cfg, adapters, ks)
	// Enable leak detection: explicit config or auto-enable when any security feature is on.
	leakEnabled := cfg.Security.LeakDetection.Enabled ||
		cfg.Security.KillSwitch.Enabled ||
		cfg.Security.DNSGuard.Enabled
	if leakEnabled {
		leak = security.NewLeakTester(
			parseDuration(cfg.Security.LeakDetection.Interval, 60*time.Second),
			cfg.Security.IntegrityCheck.ExpectedIPs,
		)
	}

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
	watchdog.ConfigureFailover(profilePriorities())
	watchdog.SetEventCallback(func(profile, event string, attempt int, err error) {
		switch event {
		case "reconnect_attempt":
			emitAlert("WARN", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
			logAudit("WARN", "vpn.reconnect_attempt", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
		case "reconnect_failed":
			emitAlert("ERROR", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
			logAudit("ERROR", "vpn.reconnect_failed", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
			if svc := profileSvc.Load(); svc != nil {
				svc.HandleReconnectError(profile, err)
			}
		case "reconnected":
			if col != nil {
				col.RecordReconnect(profile)
			}
			emitAlert("INFO", profile, cf("cli.alert.reconnected", profile, attempt))
			logAudit("INFO", "vpn.reconnected", profile, cf("cli.alert.reconnected", profile, attempt))
		case "health_degraded":
			emitAlert("WARN", profile, cf("cli.alert.health_degraded", profile, err))
			logAudit("WARN", "vpn.health_degraded", profile, cf("cli.alert.health_degraded", profile, err))
		case "failover_started":
			emitAlert("WARN", profile, cf("cli.alert.failover_started", profile))
			logAudit("WARN", "vpn.failover_started", profile, cf("cli.alert.failover_started", profile))
		case "failover_failed":
			emitAlert("ERROR", profile, cf("cli.alert.failover_failed", profile, err))
			logAudit("ERROR", "vpn.failover_failed", profile, cf("cli.alert.failover_failed", profile, err))
		case "failover_activated":
			emitAlert("INFO", profile, cf("cli.alert.failover_activated", profile))
			logAudit("INFO", "vpn.failover_activated", profile, cf("cli.alert.failover_activated", profile))
		}
	})

	// DNS manager — reference-counted guard across simultaneous tunnels.
	dnsGuard := security.NewDNSGuard()
	dnsMgr = monitor.NewDNSManager(dnsGuard, log)

	// Policy resolver — background DNS re-resolution for domain-based split tunnel.
	policyResolver = monitor.NewPolicyResolver(cfg, routeMgr, log)
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
