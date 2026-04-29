// Command kongtrol is the VPN Kongtrol CLI.
// It orchestrates multiple VPN connections, controls traffic routing,
// enforces security policies, and exposes a monitoring dashboard.
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

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"

	// Adapter registrations — order is irrelevant; all run via init().
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/ciscoanyconnect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/cloudflarewarp"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/globalprotect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/tailscale"
	// wireguard adapter is imported by name above for ParseEndpoint etc.
	// Its init() registers the adapter automatically.
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
	col             *monitor.Collector
	watchdog        *monitor.Watchdog
	dnsMgr          *monitor.DNSManager
	policyResolver  *monitor.PolicyResolver
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
	rootCmd.AddCommand(mapCmd)
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

		// Write PID so `kongtrol down` can stop this daemon.
		writePIDFile()
		defer removePIDFile()

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

		// Activate kill switch now that tunnels are up.
		if ks != nil && cfg.Security.KillSwitch.Enabled {
			// Use the first connected tunnel's interface for allowed traffic.
			tunnelIface := ""
			for _, name := range targets {
				if info, err := adapters[name].TunnelInfo(); err == nil && info != nil && info.InterfaceName != "" {
					tunnelIface = info.InterfaceName
					break
				}
			}
			if err := ks.Enable(tunnelIface, cfg.Security.KillSwitch.AllowLAN); err != nil {
				fmt.Fprintf(os.Stderr, "warning: kill switch enable failed: %v\n", err)
			}
			defer func() {
				_ = ks.Disable()
			}()
		}

		// Start watchdog after all profiles are up.
		if watchdog != nil {
			watchdog.Start(ctx)
			defer watchdog.Stop()
		}

		// Start background DNS resolver for domain-based split tunneling.
		if policyResolver != nil {
			policyResolver.Start(ctx)
			defer policyResolver.Stop()
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
			if policyResolver != nil {
				policyResolver.UnregisterProfile(name)
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
			stopDaemon()
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
		stopDaemon()
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

// ── map ──────────────────────────────────────────────────────────────────────

var mapCmd = &cobra.Command{
	Use:   "map [target]",
	Short: "Show traffic routing map — which VPN handles each destination",
	Long:  "Display all policy rules and their resolved IPs. Optionally query a specific IP or domain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if engine == nil {
			return fmt.Errorf("policy engine not loaded")
		}

		// If a target is given, resolve it and print the result.
		if len(args) > 0 {
			printResolve(args[0])
			fmt.Println()
		}

		printTrafficMap()
		return nil
	},
}

func printResolve(target string) {
	ip := net.ParseIP(target)

	if ip != nil {
		vpnName, matched := engine.ResolveIP(ip)
		if matched {
			fmt.Printf("  %s %s → %s\n",
				paint(cSuccess, "●"),
				paintBold(cBright, target),
				paintBold(cInfo, vpnName))
		} else {
			fmt.Printf("  %s %s → %s\n",
				paint(cDim, "○"),
				paintBold(cBright, target),
				paint(cDim, "default route (no matching policy)"))
		}
	} else {
		vpnName, matched := engine.ResolveDomain(target)
		if matched {
			fmt.Printf("  %s %s → %s\n",
				paint(cSuccess, "●"),
				paintBold(cBright, target),
				paintBold(cInfo, vpnName))
		} else {
			fmt.Printf("  %s %s → %s\n",
				paint(cDim, "○"),
				paintBold(cBright, target),
				paint(cDim, "default route (no matching policy)"))
		}
	}
}

func printTrafficMap() {
	rules := engine.Rules()
	if len(rules) == 0 {
		fmt.Println("  No policies configured.")
		return
	}

	// Get resolved IPs from PolicyResolver if available.
	var resolvedByProfile map[string]int
	if policyResolver != nil {
		snapshots := policyResolver.Snapshot()
		resolvedByProfile = make(map[string]int, len(snapshots))
		for _, snap := range snapshots {
			resolvedByProfile[snap.Name] = len(snap.ResolvedCIDRs)
		}
	}

	// Calculate column widths.
	nameW, matchW, viaW := 16, 28, 14
	for _, r := range rules {
		if l := len(r.Name); l+2 > nameW {
			nameW = l + 2
		}
		m := summarizeMatch(&r)
		if l := len(m); l+2 > matchW {
			matchW = l + 2
		}
		if l := len(r.Via); l+2 > viaW {
			viaW = l + 2
		}
	}
	// Cap match column.
	if matchW > 40 {
		matchW = 40
	}

	// Header.
	hdr := fmt.Sprintf("  %-*s %-*s %-*s %s",
		nameW, "POLICY", matchW, "MATCH", viaW, "VIA", "RESOLVED")
	fmt.Println(paint(cDim, hdr))
	fmt.Println(paint(cDim, "  "+strings.Repeat("─", nameW+matchW+viaW+12)))

	// Rows.
	for _, r := range rules {
		match := summarizeMatch(&r)
		if len(match) > matchW {
			match = match[:matchW-1] + "…"
		}

		resolved := paint(cDim, "—")
		if resolvedByProfile != nil {
			if n, ok := resolvedByProfile[r.Via]; ok && n > 0 {
				resolved = paint(cSuccess, fmt.Sprintf("%d IPs", n))
			}
		}

		// Color the match column: domains in blue, IPs in yellow.
		matchColored := match
		if len(r.Match.Domains) > 0 && len(r.Match.IPRanges) == 0 {
			matchColored = paint(cInfo, match)
		} else if len(r.Match.IPRanges) > 0 && len(r.Match.Domains) == 0 {
			matchColored = paint(cWarn, match)
		}

		fmt.Printf("  %-*s %-*s %-*s %s\n",
			nameW, paintBold(cBright, r.Name),
			matchW, matchColored,
			viaW, paintBold(cInfo, r.Via),
			resolved)
	}
}

func summarizeMatch(r *policy.Rule) string {
	var parts []string
	for _, d := range r.Match.Domains {
		parts = append(parts, d)
	}
	for _, n := range r.Match.IPRanges {
		parts = append(parts, n.String())
	}
	return strings.Join(parts, ", ")
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
				Extra:      map[string]string{"protocol": vpnCfg.Protocol},
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
	dnsMgr = monitor.NewDNSManager(dnsGuard, log)

	// Policy resolver — background DNS re-resolution for domain-based split tunnel.
	policyResolver = monitor.NewPolicyResolver(cfg, routeMgr, log)

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

	// For WireGuard: if policies constrain this profile to specific
	// destinations, rewrite AllowedIPs in the config so only policy-matched
	// traffic flows through the tunnel (split tunnel via WireGuard's own mechanism).
	if vpnCfg.Type == "wireguard" && aCfg.ConfigPath != "" {
		if cidrs := policyAllowedIPs(name); len(cidrs) > 0 {
			// Preserve the tunnel name derived from the ORIGINAL config path so
			// that interfaceFromConfig on the temp file does not produce a wrong name.
			if aCfg.TunnelName == "" {
				base := filepath.Base(aCfg.ConfigPath)
				aCfg.TunnelName = strings.TrimSuffix(base, filepath.Ext(base))
			}
			patched, err := patchWireGuardAllowedIPs(aCfg.ConfigPath, aCfg.TunnelName, cidrs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "policy: failed to patch WireGuard AllowedIPs for %s, using original config: %v\n", name, err)
			} else {
				aCfg.ConfigPath = patched
				defer os.RemoveAll(filepath.Dir(patched))
			}
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
	// Skip for WireGuard: it configures DNS natively via the [Interface] DNS
	// field in the .conf file. Applying netsh on top causes errors and is
	// redundant — especially in split-tunnel mode where the WireGuard iface
	// does not carry general DNS traffic.
	if dnsMgr != nil && cfg.Security.DNSGuard.Enabled && vpnCfg.Type != "wireguard" {
		if info, err := adapter.TunnelInfo(); err == nil && info != nil && len(info.DNS) > 0 {
			dnsMgr.OnConnect(name, info.InterfaceName, info.DNS)
		}
	}

	// Register with PolicyResolver for dynamic DNS-based split tunneling.
	// Uses the ORIGINAL config path (not the temp patched one) to parse
	// peer key and endpoint — the temp file may already be deleted.
	if policyResolver != nil && vpnCfg.Type == "wireguard" {
		ifaceName := interfaceFromWGConfig(vpnCfg)
		if err := policyResolver.RegisterProfile(name, ifaceName, vpnCfg.ConfigFile); err != nil {
			fmt.Fprintf(os.Stderr, "policyresolver: %s: %v\n", name, err)
		}
	}

	return nil
}

// interfaceFromWGConfig derives the WireGuard interface name from a VPN config.
func interfaceFromWGConfig(vpnCfg config.VPNConfig) string {
	if vpnCfg.TunnelName != "" {
		return vpnCfg.TunnelName
	}
	base := filepath.Base(vpnCfg.ConfigFile)
	return strings.TrimSuffix(base, filepath.Ext(base))
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
		engine,
		policyResolver,
		dnsMgr,
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

// policyAllowedIPs collects all IP CIDRs from policies that route via profileName.
// Includes essential CIDRs (VPN subnet, DNS) and excludes the WireGuard endpoint.
// Returns nil when no policies constrain this profile (full-tunnel mode).
func policyAllowedIPs(profileName string) []string {
	vpnCfg, ok := cfg.VPNs[profileName]
	if !ok {
		return nil
	}

	seen := make(map[string]struct{})
	var cidrs []string
	add := func(cidr string) {
		if cidr == "" {
			return
		}
		if _, ok := seen[cidr]; !ok {
			seen[cidr] = struct{}{}
			cidrs = append(cidrs, cidr)
		}
	}

	// Parse WireGuard config for essential CIDRs and endpoint exclusion.
	var endpointIP net.IP
	if vpnCfg.ConfigFile != "" {
		endpointIP, _ = wireguard.ParseEndpoint(vpnCfg.ConfigFile)

		// VPN subnet: 10.2.0.2/32 → 10.2.0.0/24
		if addr, err := wireguard.ParseConfigAddress(vpnCfg.ConfigFile); err == nil && addr != nil {
			if v4 := addr.To4(); v4 != nil {
				add(fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2]))
			}
		}

		// DNS servers must route through tunnel.
		for _, dns := range wireguard.ParseConfigDNS(vpnCfg.ConfigFile) {
			if dns.To4() != nil {
				add(dns.String() + "/32")
			}
		}
	}

	hasDomains := false
	for _, pol := range cfg.Policies {
		if pol.Via != profileName {
			continue
		}
		for _, cidr := range pol.Match.IPRanges {
			add(cidr)
		}
		for _, domain := range pol.Match.Domains {
			hasDomains = true
			// Expand wildcards to probe common subdomains.
			var lookups []string
			if strings.HasPrefix(domain, "*.") {
				base := strings.TrimPrefix(domain, "*.")
				lookups = append(lookups, base)
				for _, prefix := range []string{"www.", "api.", "cdn.", "app.", "docs.", "console."} {
					lookups = append(lookups, prefix+base)
				}
			} else {
				lookups = []string{domain}
			}

			for _, lookup := range lookups {
				ips, err := net.LookupHost(lookup)
				if err != nil {
					continue // subdomain may not exist — not an error
				}
				for _, ip := range ips {
					parsed := net.ParseIP(ip)
					if parsed == nil {
						continue
					}
					// Exclude endpoint IP to prevent routing loops.
					if endpointIP != nil && parsed.Equal(endpointIP) {
						continue
					}
					if parsed.To4() != nil {
						add(ip + "/32")
					}
					// Skip IPv6 for initial AllowedIPs — simplifies routing.
				}
			}
		}
	}

	if !hasDomains && len(cidrs) == 0 {
		return nil
	}
	return cidrs
}

// patchWireGuardAllowedIPs writes a copy of the WireGuard config with all
// [Peer] AllowedIPs entries replaced by cidrs.
// tunnelName must match the original config basename (without .conf) so that
// WireGuard derives the correct interface name from the temp file.
func patchWireGuardAllowedIPs(configPath, tunnelName string, cidrs []string) (string, error) {
	original, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}

	allowedLine := "AllowedIPs = " + strings.Join(cidrs, ", ")
	var out strings.Builder
	inPeer := false
	replaced := false
	for _, line := range strings.Split(string(original), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPeer = strings.EqualFold(trimmed, "[Peer]")
			replaced = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(trimmed), "allowedips") {
			if !replaced {
				out.WriteString(allowedLine + "\n")
				replaced = true
			}
			// skip original AllowedIPs line
			continue
		}
		out.WriteString(line + "\n")
	}

	// The temp dir must contain a file named <tunnelName>.conf so WireGuard
	// creates an interface with the correct name.
	tmpDir, err := os.MkdirTemp("", "kongtrol-wg-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	tmpPath := filepath.Join(tmpDir, tunnelName+".conf")
	if err := os.WriteFile(tmpPath, []byte(out.String()), 0600); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("write temp config: %w", err)
	}
	return tmpPath, nil
}

func openBrowser(url string) {
	// Platform-specific browser open is handled at runtime.
	fmt.Printf("Open: %s\n", url)
}

// ── PID file helpers ──────────────────────────────────────────────────────────

// pidFilePath returns ~/.kongtrol/run/kongtrol.pid.
func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kongtrol", "run", "kongtrol.pid")
}

// writePIDFile records the current process PID so that a concurrent
// `kongtrol down` invocation can stop this daemon.
func writePIDFile() {
	path := pidFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0600)
}

// removePIDFile removes the PID file only if it still holds our own PID
// (guards against a race where a new `kongtrol up` has already replaced it).
func removePIDFile() {
	path := pidFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid == os.Getpid() {
		_ = os.Remove(path)
	}
}

// stopDaemon reads the PID file and terminates the running `kongtrol up`
// daemon so it does not attempt to reconnect profiles that were just brought
// down.
func stopDaemon() {
	path := pidFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // no daemon running
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid == os.Getpid() {
		return // stale or own PID
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: cannot find daemon process %d: %v\n", pid, err)
		return
	}
	if err := proc.Kill(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: cannot stop daemon process %d: %v (try running as Administrator)\n", pid, err)
		return
	}
	_ = os.Remove(path)
	fmt.Fprintf(os.Stderr, "[*] stopped background daemon (pid %d)\n", pid)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
