// Command kongtrol is the VPN Kongtrol CLI.
// It orchestrates multiple VPN connections, controls traffic routing,
// enforces security policies, and exposes a monitoring dashboard.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	// Adapter registrations — order is irrelevant; all run via init().
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/ciscoanyconnect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/cloudflarewarp"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/globalprotect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/tailscale"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3".
// Falls back to "dev" when built without ldflags (local go build).
var version = "dev"

var (
	cfgPath  string
	cfg      *config.Config
	adapters map[string]vpn.VPNAdapter
	routeMgr routing.RouteManager
	engine   *policy.Engine
	ks       security.KillSwitch
	leak     *security.LeakTester
	audit    *security.AuditLogger
	col      *monitor.Collector
	watchdog *monitor.Watchdog
	dnsMgr   *monitor.DNSManager
)

var rootCmd = &cobra.Command{
	Use:   "kongtrol",
	Short: "Multi-VPN orchestration — route traffic, enforce security, monitor tunnels",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// init shows its own animated logo — skip the compact header there.
		if cmd.Name() != "init" {
			PrintHeader(version)
		}
		// Skip config load for init and version commands.
		if cmd.Name() == "init" || cmd.Name() == "version" {
			return nil
		}
		return loadConfig()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path (default: ~/.kongtrol/kongtrol.yaml)")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(routesCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(exportCmd)
}

// ── up ───────────────────────────────────────────────────────────────────────

var upGroup string

var upCmd = &cobra.Command{
	Use:   "up [profile...]",
	Short: "Connect one or more VPN profiles (or a group with --group)",
	RunE: func(cmd *cobra.Command, args []string) error {
		targets, err := resolveProfiles(args, upGroup)
		if err != nil {
			return err
		}
		ctx := contextWithSignal()

		// Restore DNS on any exit (SIGTERM, panic path).
		defer func() {
			if dnsMgr != nil {
				dnsMgr.ForceRestore()
			}
		}()

		for _, name := range targets {
			if err := connectProfile(ctx, name); err != nil {
				return fmt.Errorf("up %s: %w", name, err)
			}
			fmt.Printf("[+] %s connected\n", name)
		}

		// Start watchdog after all profiles are up.
		if watchdog != nil {
			watchdog.Start(ctx)
			defer watchdog.Stop()
		}

		// Block until signal.
		<-ctx.Done()
		return nil
	},
}

func init() {
	upCmd.Flags().StringVar(&upGroup, "group", "", "connect all profiles in a named group")
}

// ── down ─────────────────────────────────────────────────────────────────────

var downAll bool
var downGroup string

var downCmd = &cobra.Command{
	Use:   "down [profile...]",
	Short: "Disconnect one or more VPN profiles (or a group with --group)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := contextWithSignal()
		disconnect := func(name string) error {
			adapter, ok := adapters[name]
			if !ok {
				return fmt.Errorf("unknown profile %q", name)
			}
			if watchdog != nil {
				watchdog.MarkIntended(name)
			}
			if err := adapter.Disconnect(ctx); err != nil {
				return err
			}
			if dnsMgr != nil {
				dnsMgr.OnDisconnect(name)
			}
			return nil
		}

		if downAll {
			for name := range adapters {
				if err := disconnect(name); err != nil {
					fmt.Fprintf(os.Stderr, "[-] %s: %v\n", name, err)
				} else {
					fmt.Printf("[-] %s disconnected\n", name)
				}
			}
			return nil
		}

		targets, err := resolveProfiles(args, downGroup)
		if err != nil {
			return err
		}
		for _, name := range targets {
			if err := disconnect(name); err != nil {
				return fmt.Errorf("down %s: %w", name, err)
			}
			fmt.Printf("[-] %s disconnected\n", name)
		}
		return nil
	},
}

func init() {
	downCmd.Flags().BoolVar(&downAll, "all", false, "disconnect all active tunnels")
	downCmd.Flags().StringVar(&downGroup, "group", "", "disconnect all profiles in a named group")
}

// ── status ───────────────────────────────────────────────────────────────────

var statusWatch bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all VPN tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusWatch {
			return runStatusWatch()
		}
		printStatus()
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "auto-refresh every 2 seconds")
}

func printStatus() {
	fmt.Printf("%-16s %-14s %-18s %s\n", "PROFILE", "STATUS", "IP", "UPTIME")
	fmt.Printf("%-16s %-14s %-18s %s\n", "-------", "------", "--", "------")

	for name, adapter := range adapters {
		status := adapter.Status()
		ip := "—"
		uptime := "—"

		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			if info.AssignedIP != nil {
				ip = info.AssignedIP.String()
			}
			if !info.ConnectedAt.IsZero() {
				uptime = time.Since(info.ConnectedAt).Round(time.Second).String()
			}
		}

		fmt.Printf("%-16s %-14s %-18s %s\n", name, status, ip, uptime)
	}

	if ks != nil {
		ksStatus := "OFF"
		if ks.IsEnabled() {
			ksStatus = "ON"
		}
		fmt.Printf("\nKill Switch: %s\n", ksStatus)
	}
}

func runStatusWatch() error {
	ctx := contextWithSignal()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	clearScreen()
	printStatus()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			clearScreen()
			fmt.Printf("Last updated: %s  (Ctrl+C to exit)\n\n", time.Now().Format("15:04:05"))
			printStatus()
		}
	}
}

func clearScreen() {
	// ANSI escape: move cursor to top-left and clear screen.
	fmt.Print("\033[H\033[2J")
}

// ── routes ───────────────────────────────────────────────────────────────────

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Manage routing rules",
}

var routesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active Kongtrol-managed routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		routes, err := routeMgr.List()
		if err != nil {
			return err
		}
		if len(routes) == 0 {
			fmt.Println("No routes managed by kongtrol.")
			return nil
		}
		fmt.Printf("%-24s %-18s %-12s %s\n", "DESTINATION", "GATEWAY", "INTERFACE", "METRIC")
		for _, r := range routes {
			gw := "—"
			if r.Gateway != nil {
				gw = r.Gateway.String()
			}
			fmt.Printf("%-24s %-18s %-12s %d\n", r.Destination.String(), gw, r.Interface, r.Metric)
		}
		return nil
	},
}

func init() {
	routesCmd.AddCommand(routesListCmd)
}

// ── check ─────────────────────────────────────────────────────────────────────

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run integrity and leak tests immediately",
	RunE: func(cmd *cobra.Command, args []string) error {
		if leak == nil {
			return fmt.Errorf("leak tester not initialized (enable monitor in config)")
		}
		fmt.Println("Running leak test…")
		result := leak.CheckNow()
		if result.HasLeak {
			fmt.Printf("[LEAK] %s\n", result.Reason)
			return fmt.Errorf("leak detected")
		}
		fmt.Printf("[OK] No leak detected. Public IP: %s\n", result.PublicIP)
		return nil
	},
}

// ── dashboard ─────────────────────────────────────────────────────────────────

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web dashboard and open it in your browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := buildAPIServer()
		if err := srv.Start(); err != nil {
			return fmt.Errorf("dashboard: %w", err)
		}
		addr := srv.Addr()
		fmt.Printf("Dashboard running at %s\n", addr)
		openBrowser(addr)

		// Block until Ctrl+C.
		<-contextWithSignal().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	},
}

// ── audit ─────────────────────────────────────────────────────────────────────

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Manage the audit log",
}

// ── config ────────────────────────────────────────────────────────────────────

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config management",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate kongtrol.yaml without connecting",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := config.Load(cfgPath); err != nil {
			return err
		}
		fmt.Println("[OK] Config is valid.")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}

// ── export ────────────────────────────────────────────────────────────────────

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print a sanitized config template (no secrets) for sharing with teammates",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no config loaded")
		}
		fmt.Println("# Kongtrol config template — generated by 'kongtrol export'")
		fmt.Println("# Passwords and keys are redacted. Run 'kongtrol init' on the target machine")
		fmt.Println("# to store credentials in the OS keychain.")
		fmt.Println()
		fmt.Println("vpns:")
		for name, v := range cfg.VPNs {
			fmt.Printf("  %s:\n", name)
			fmt.Printf("    type: %s\n", v.Type)
			if v.Version != "" {
				fmt.Printf("    version: %q\n", v.Version)
			}
			if v.Host != "" {
				fmt.Printf("    host: %s\n", v.Host)
			}
			if v.Port != 0 {
				fmt.Printf("    port: %d\n", v.Port)
			}
			if v.TunnelName != "" {
				fmt.Printf("    tunnel_name: %q\n", v.TunnelName)
			}
			if v.ConfigFile != "" {
				fmt.Printf("    config: %s\n", v.ConfigFile)
			}
			if v.Server != "" {
				fmt.Printf("    server: %s\n", v.Server)
			}
			if v.Protocol != "" {
				fmt.Printf("    protocol: %s\n", v.Protocol)
			}
			if v.Priority != 0 {
				fmt.Printf("    priority: %d\n", v.Priority)
			}
			fmt.Printf("    auth:\n")
			fmt.Printf("      method: %s\n", v.Auth.Method)
			if v.Auth.Cert != "" {
				fmt.Printf("      cert: %s\n", v.Auth.Cert)
			}
			if v.Auth.Key != "" {
				fmt.Printf("      key: %s\n", v.Auth.Key)
			}
			if v.Auth.Username != "" {
				fmt.Printf("      username: %s\n", v.Auth.Username)
			}
			if v.Auth.PasswordKeychain != "" {
				fmt.Printf("      password_keychain: %s  # store via: kongtrol init\n", v.Auth.PasswordKeychain)
			}
			if v.Auth.UsernameKeychain != "" {
				fmt.Printf("      username_keychain: %s  # store via: kongtrol init\n", v.Auth.UsernameKeychain)
			}
		}
		if len(cfg.Policies) > 0 {
			fmt.Println("\npolicies:")
			for _, p := range cfg.Policies {
				fmt.Printf("  - name: %q\n", p.Name)
				if len(p.Match.IPRanges) > 0 {
					fmt.Printf("    match:\n      ip_ranges: %v\n", p.Match.IPRanges)
				}
				if len(p.Match.Domains) > 0 {
					fmt.Printf("    match:\n      domains: %v\n", p.Match.Domains)
				}
				fmt.Printf("    via: %s\n", p.Via)
			}
		}
		if len(cfg.Groups) > 0 {
			fmt.Println("\ngroups:")
			for name, g := range cfg.Groups {
				fmt.Printf("  %s:\n    profiles: %v\n", name, g.Profiles)
			}
		}
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveProfiles returns the list of profile names from explicit args or a group.
func resolveProfiles(args []string, group string) ([]string, error) {
	if group != "" {
		if cfg == nil {
			return nil, fmt.Errorf("no config loaded")
		}
		g, ok := cfg.Groups[group]
		if !ok {
			return nil, fmt.Errorf("unknown group %q — check 'groups:' in your config", group)
		}
		return g.Profiles, nil
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("specify at least one profile name or use --group")
	}
	return args, nil
}

func loadConfig() error {
	var err error
	cfg, err = config.Load(cfgPath)
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
	if cfg.Security.LeakDetection.Enabled {
		leak = security.NewLeakTester(
			parseDuration(cfg.Security.LeakDetection.Interval, 60*time.Second),
			cfg.Security.IntegrityCheck.ExpectedIPs,
		)
	}

	// Monitor.
	col = monitor.NewCollector(adapters)
	col.Start(5 * time.Second)

	// Watchdog — auto-reconnect on unexpected disconnects.
	log, _ := zap.NewProduction()
	watchdog = monitor.NewWatchdog(adapters, connectProfile, log)

	// DNS manager — reference-counted guard across simultaneous tunnels.
	dnsGuard := security.NewDNSGuard()
	dnsMgr = monitor.NewDNSManager(dnsGuard, "", log)

	return nil
}

func connectProfile(ctx context.Context, name string) error {
	adapter, ok := adapters[name]
	if !ok {
		return fmt.Errorf("unknown profile %q", name)
	}

	vpnCfg, ok := cfg.VPNs[name]
	if !ok {
		return fmt.Errorf("no config for profile %q", name)
	}

	aCfg := vpn.AdapterConfig{
		Host:       vpnCfg.Host,
		Port:       vpnCfg.Port,
		TunnelName: vpnCfg.TunnelName,
		CertPath:   vpnCfg.Auth.Cert,
		KeyPath:    vpnCfg.Auth.Key,
		ConfigPath: vpnCfg.ConfigFile,
		Username:   vpnCfg.Auth.Username,
		Extra:      map[string]string{"protocol": vpnCfg.Protocol},
	}

	// Resolve credentials from OS keychain.
	if vpnCfg.Auth.UsernameKeychain != "" {
		u, err := config.GetCredential(name, "username")
		if err == nil {
			aCfg.Username = u
		}
	}
	if vpnCfg.Auth.PasswordKeychain != "" {
		p, err := config.GetCredential(name, "password")
		if err == nil {
			aCfg.Password = p
		}
	}

	if err := adapter.Connect(ctx, aCfg); err != nil {
		return err
	}

	// Mark active so watchdog monitors this profile for unexpected drops.
	if watchdog != nil {
		watchdog.MarkActive(name)
	}

	// Wire DNS guard if tunnel published DNS servers.
	if dnsMgr != nil && cfg.Security.DNSGuard.Enabled {
		if info, err := adapter.TunnelInfo(); err == nil && info != nil && len(info.DNS) > 0 {
			dnsMgr.OnConnect(name, info.DNS)
		}
	}

	return nil
}

func buildAPIServer() *api.Server {
	return api.NewServer(
		cfg.Monitor.Dashboard.Bind,
		cfg.Monitor.Dashboard.Port,
		adapters,
		col,
		routeMgr,
		ks,
		leak,
	)
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

func openBrowser(url string) {
	// Platform-specific browser open is handled at runtime.
	fmt.Printf("Open: %s\n", url)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
