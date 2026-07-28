package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func emitAlert(level, profile, message string) {
	if outputQuiet {
		return
	}
	if alertBell {
		fmt.Print("\a")
	}
	var rendered string
	switch level {
	case "ERROR":
		rendered = tuiErr(message)
	case "WARN":
		rendered = tuiWarn(message)
	default:
		rendered = tuiInfo(message)
	}
	// While the up/status Bubble Tea view owns the terminal (alt screen),
	// route alerts into its scrollable log panel instead of stderr —
	// writing directly here would corrupt the alt-screen display.
	if sendTUILog(rendered) {
		return
	}
	fmt.Fprintln(os.Stderr, rendered)
}

func logAudit(level, action, profile, message string) {
	if audit == nil {
		return
	}
	_ = audit.Log(level, action, profile, message)
}

func buildProfileService() *app.ProfileService {
	return app.NewProfileService(app.ProfileServiceDeps{
		Cfg:            cfg,
		Adapters:       adapters,
		Watchdog:       watchdog,
		PolicyResolver: policyResolver,
		DNSManager:     dnsMgr,
		ApplyKill:      applyKillSwitchState,
		Format:         cf,
		Translate:      ct,
		EmitAlert:      emitAlert,
		LogAudit:       logAudit,
		EmitWarn: func(message string) {
			fmt.Fprintln(os.Stderr, tuiWarn(message))
		},
		EmitStderr: func(message string) {
			fmt.Fprintln(os.Stderr, message)
		},
	})
}

func disconnectProfile(ctx context.Context, name string) error {
	svc := profileSvc.Load()
	if svc == nil {
		return fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
	}
	return svc.Disconnect(ctx, name)
}

func connectProfile(ctx context.Context, name string) error {
	svc := profileSvc.Load()
	if svc == nil {
		return fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
	}
	return svc.Connect(ctx, name)
}

// buildAPIServer wires up the embedded API/dashboard server. onShutdown is
// called (in its own goroutine, by the server) when a client POSTs
// /api/v1/shutdown — it should cancel whatever context this process is
// blocking on, so its own deferred cleanup runs, exactly like Ctrl+C or
// SIGTERM. Pass nil to leave the endpoint disabled (501) for a server
// instance with no well-defined owning context to cancel.
func buildAPIServer(onShutdown func()) *api.Server {
	return api.NewServer(
		cfg.Monitor.Dashboard.Bind,
		cfg.Monitor.Dashboard.Port,
		adapters,
		col,
		routeMgr,
		ks,
		cfg.Security.KillSwitch.Enabled,
		leak,
		engine,
		policyResolver,
		activeCfgPath,
		func(newCfg *config.Config, newEngine *policy.Engine) {
			cfg = newCfg
			engine = newEngine
			killSwitchSvc = app.NewKillSwitchService(cfg, adapters, ks)
			profileSvc.Store(buildProfileService())
		},
		func(newCfg *config.Config) {
			_ = applyKillSwitchState()
			applyDNSGuardState()
		},
		dnsMgr,
		cfg.Security.DNSGuard.Enabled,
		connectProfile,
		disconnectProfile,
		apiToken,
		onShutdown,
	)
}

func resolveConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	for _, candidate := range config.DefaultPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s", ct("cli.error.config_not_found"))
}

func contextWithSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()
	return ctx
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func metricsHistoryPath() string {
	if cfg != nil && strings.TrimSpace(cfg.Monitor.History.Path) != "" {
		return cfg.Monitor.History.Path
	}
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		return "kongtrol-history.json"
	}
	return filepath.Join(home, ".kongtrol", "history.json")
}

func startHistoryPersistence(ctx context.Context) {
	if col == nil {
		return
	}
	flushEvery := parseDuration(cfg.Monitor.History.FlushInterval, 30*time.Second)
	if flushEvery <= 0 {
		flushEvery = 30 * time.Second
	}
	path := metricsHistoryPath()
	go func() {
		t := time.NewTicker(flushEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = col.SaveHistory(path)
				return
			case <-t.C:
				_ = col.SaveHistory(path)
			}
		}
	}()
}

func profilePriorities() map[string]int {
	out := make(map[string]int, len(cfg.VPNs))
	for name, vpnCfg := range cfg.VPNs {
		if vpnCfg.Priority > 0 {
			out[name] = vpnCfg.Priority
		}
	}
	return out
}

func healthCheckProfile(ctx context.Context, name string, adapter vpn.VPNAdapter) monitor.HealthResult {
	res := monitor.HealthResult{
		CheckedAt: time.Now(),
	}
	if adapter.Status().Normalize() != vpn.StatusConnected {
		res.Error = "tunnel is not connected"
		return res
	}

	info, err := adapter.TunnelInfo()
	if err != nil {
		res.Error = err.Error()
		return res
	}

	host := cfg.VPNs[name].Host
	port := cfg.VPNs[name].Port
	if port == 0 {
		port = 443
	}
	if host == "" && info != nil && info.RemoteIP != nil {
		host = info.RemoteIP.String()
	}
	if host != "" {
		started := time.Now()
		dialer := net.Dialer{}
		conn, dialErr := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		res.Latency = time.Since(started)
		if dialErr == nil {
			res.Reachable = true
			_ = conn.Close()
		}
	} else {
		res.Reachable = true
	}

	// Validate DNS resolution through tunnel DNS when available.
	res.DNSOK = true
	if info != nil && len(info.DNS) > 0 {
		res.DNSOK = false
		hostname := "connectivitycheck.gstatic.com"
		for _, dns := range info.DNS {
			dns := dns
			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(c context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{}
					return d.DialContext(c, "udp", net.JoinHostPort(dns.String(), "53"))
				},
			}
			if addrs, lookupErr := resolver.LookupIPAddr(ctx, hostname); lookupErr == nil && len(addrs) > 0 {
				res.DNSOK = true
				break
			}
		}
	}

	res.Healthy = res.Reachable && res.DNSOK
	if !res.Healthy {
		res.Error = fmt.Sprintf("reachable=%t dns_ok=%t", res.Reachable, res.DNSOK)
	}
	if col != nil {
		col.RecordHealth(name, res.Latency, res.Reachable, res.DNSOK)
	}
	return res
}

func applyKillSwitchState() error {
	if killSwitchSvc == nil {
		return nil
	}
	if err := killSwitchSvc.Apply(); err != nil {
		fmt.Fprintln(os.Stderr, cf("cli.up.warn.kill_switch_enable", err))
		return err
	}
	return nil
}

// applyDNSGuardState re-evaluates DNS guard enforcement against the current
// cfg.Security.DNSGuard.Enabled setting for every already-connected profile.
// It mirrors the enable gate in ProfileService.Connect (internal/app/profile_service.go)
// so that flipping the setting at runtime (via the dashboard toggle) takes
// effect immediately instead of only on the next connect/disconnect.
func applyDNSGuardState() {
	if dnsMgr == nil || cfg == nil {
		return
	}
	if !cfg.Security.DNSGuard.Enabled {
		dnsMgr.ForceRestore()
		return
	}
	for name, a := range adapters {
		vpnCfg, ok := cfg.VPNs[name]
		if !ok || vpnCfg.Type == "wireguard" {
			continue
		}
		if a.Status().Normalize() != vpn.StatusConnected {
			continue
		}
		info, err := a.TunnelInfo()
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

func policyAllowedIPs(profileName string) []string {
	svc := profileSvc.Load()
	if svc == nil {
		return nil
	}
	return svc.PolicyAllowedIPs(profileName)
}

// openBrowser launches the OS default browser at url. Callers decide whether
// and how to report success/failure — this never prints, since it's also
// called from inside the Bubble Tea TUI where a direct print would corrupt
// the alt-screen display.
func openBrowser(url string) error {
	return launchBrowser(url)
}
