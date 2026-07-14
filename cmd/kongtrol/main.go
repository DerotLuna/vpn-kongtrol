// Command kongtrol is the VPN Kongtrol CLI.
// It orchestrates multiple VPN connections, controls traffic routing,
// enforces security policies, and exposes a monitoring dashboard.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
	"go.uber.org/zap"

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
// Falls back to a local dev marker when built without ldflags.
var version = "v1.0.1-dev"

var (
	cfgPath        string
	activeCfgPath  string
	cfg            *config.Config
	adapters       map[string]vpn.VPNAdapter
	routeMgr       routing.RouteManager
	engine         *policy.Engine
	ks             security.KillSwitch
	leak           *security.LeakTester
	audit          *security.AuditLogger
	col            *monitor.Collector
	watchdog       *monitor.Watchdog
	dnsMgr         *monitor.DNSManager
	policyResolver *monitor.PolicyResolver
)

var rootCmd = &cobra.Command{
	Use:   "kongtrol",
	Short: "Multi-VPN orchestration — route traffic, enforce security, monitor tunnels",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// init shows its own animated logo — skip the compact header there.
		if cmd.Name() != "init" && cmd.Name() != "version" {
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
	rootCmd.AddCommand(versionCmd)
}

// ── up ───────────────────────────────────────────────────────────────────────

var upAll bool
var upGroup string

var upCmd = &cobra.Command{
	Use:   "up [profile...]",
	Short: "Connect one or more VPN profiles (or all with --all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		startedAll := time.Now()
		var (
			targets []string
			err     error
		)
		if upAll {
			targets = make([]string, 0, len(adapters))
			for name := range adapters {
				targets = append(targets, name)
			}
		} else {
			targets, err = resolveProfiles(args, upGroup)
			if err != nil {
				return err
			}
		}
		signalCtx := contextWithSignal()
		ctx, cancelCtx := context.WithCancel(signalCtx)
		defer cancelCtx()

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
			startedProfile := time.Now()
			spin := newSpinner(fmt.Sprintf("Connecting %s", name))
			spin.Start()
			wasConnected := adapters[name].Status().Normalize() == vpn.StatusConnected
			connectCtx, cancelConnect := context.WithTimeout(ctx, 5*time.Minute)
			connectDone := make(chan error, 1)
			go func() {
				connectDone <- connectProfile(connectCtx, name)
			}()
			select {
			case err = <-connectDone:
			case <-ctx.Done():
				err = ctx.Err()
			}
			cancelConnect()
			spin.Stop()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					fmt.Println(tuiWarn("connection cancelled"))
					return nil
				}
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
					err = fmt.Errorf("connection timed out after 5m (check VPN client window/credentials, then retry)")
				}
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error() + "  " + paint(cDim, "("+time.Since(startedProfile).Round(time.Second).String()+")")))
				return fmt.Errorf("up %s: %w", name, err)
			}
			if wasConnected {
				fmt.Println(tuiInfo(paintBold(cBright, name) + "  " + paint(cDim, "already connected ("+time.Since(startedProfile).Round(time.Second).String()+")")))
			} else {
				fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, "connected in "+time.Since(startedProfile).Round(time.Second).String())))
			}
		}
		fmt.Println(tuiInfo(paint(cDim, fmt.Sprintf("All targets connected in %s", time.Since(startedAll).Round(time.Second)))))

		// Activate kill switch now that tunnels are up.
		// Collect all tunnel interfaces so the kill switch allows their traffic.
		if ks != nil && cfg.Security.KillSwitch.Enabled {
			var tunnelIfaces []string
			for _, name := range targets {
				if info, err := adapters[name].TunnelInfo(); err == nil && info != nil && info.InterfaceName != "" {
					tunnelIfaces = append(tunnelIfaces, info.InterfaceName)
				}
			}
			// Also collect VPN endpoint IPs so encrypted tunnel traffic can reach them.
			var endpointIPs []string
			for _, name := range targets {
				if vpnCfg, ok := cfg.VPNs[name]; ok && vpnCfg.Host != "" {
					if ip := net.ParseIP(vpnCfg.Host); ip != nil {
						endpointIPs = append(endpointIPs, ip.String())
					} else {
						// Host is a hostname — resolve it.
						if ips, err := net.LookupIP(vpnCfg.Host); err == nil {
							for _, ip := range ips {
								if ip.To4() != nil {
									endpointIPs = append(endpointIPs, ip.String())
								}
							}
						}
					}
				}
				// For WireGuard, endpoint is in the config file, parse it.
				if vpnCfg, ok := cfg.VPNs[name]; ok && vpnCfg.Type == "wireguard" && vpnCfg.ConfigFile != "" {
					if epIP, err := wireguard.ParseEndpoint(vpnCfg.ConfigFile); err == nil && epIP != nil {
						endpointIPs = append(endpointIPs, epIP.String())
					}
				}
			}
			// Pass first tunnel interface + endpoint IPs as comma-separated allow list.
			allowSpec := strings.Join(tunnelIfaces, ",")
			if len(endpointIPs) > 0 {
				allowSpec += "|" + strings.Join(endpointIPs, ",")
			}
			if err := ks.Enable(allowSpec, cfg.Security.KillSwitch.AllowLAN); err != nil {
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

		// Start background leak detection.
		if leak != nil {
			leak.Start(ctx, func(result security.LeakResult) {
				if result.HasLeak {
					fmt.Fprintf(os.Stderr, "[LEAK] %s (public IP: %s)\n", result.Reason, result.PublicIP)
				}
			})
			defer leak.Stop()
		}

		// Start the API server / dashboard alongside the daemon.
		var dashURL string
		srv := buildAPIServer()
		if err := srv.Start(); err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn("dashboard server: "+err.Error()))
		} else {
			dashURL = srv.Addr()
			defer func() {
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutCtx)
			}()
		}

		// Block until signal — show live daemon view on interactive terminals.
		runUpTUI(ctx, cancelCtx, adapters, ks, dnsMgr, dashURL)
		return nil
	},
}

func init() {
	upCmd.Flags().BoolVar(&upAll, "all", false, "connect all configured profiles")
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

		disconnectWithSpinner := func(name string) error {
			spin := newSpinner(fmt.Sprintf("Disconnecting %s", name))
			spin.Start()
			err := disconnect(name)
			spin.Stop()
			if err != nil {
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error()))
				return err
			}
			fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, "disconnected")))
			return nil
		}

		if downAll {
			for name := range adapters {
				_ = disconnectWithSpinner(name)
			}
			stopDaemon()
			return nil
		}

		targets, err := resolveProfiles(args, downGroup)
		if err != nil {
			return err
		}
		for _, name := range targets {
			if err := disconnectWithSpinner(name); err != nil {
				return fmt.Errorf("down %s: %w", name, err)
			}
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

var (
	styleStatusHdr  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	styleStatusName = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleStatusUp   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	styleStatusDown = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleStatusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleStatusIP   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleStatusTime = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func statusDot(s vpn.Status) string {
	switch s {
	case vpn.StatusConnected:
		return styleStatusUp.Render("●")
	case vpn.StatusConnecting:
		return styleInfo.Render("◌")
	case vpn.StatusError:
		return styleStatusErr.Render("✗")
	default:
		return styleStatusDown.Render("○")
	}
}

func statusLabel(s vpn.Status) string {
	switch s {
	case vpn.StatusConnected:
		return styleStatusUp.Render(string(s))
	case vpn.StatusConnecting:
		return styleInfo.Render(string(s))
	case vpn.StatusError:
		return styleStatusErr.Render(string(s))
	default:
		return styleStatusDown.Render(string(s))
	}
}

type apiSecurityStatus struct {
	KillSwitch        bool `json:"kill_switch"`
	KillSwitchEnabled bool `json:"kill_switch_enabled"`
	DNSGuard          bool `json:"dns_guard"`
	DNSGuardEnabled   bool `json:"dns_guard_enabled"`
}

func dashboardURL() string {
	return fmt.Sprintf("http://%s:%d", cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port)
}

func fetchDaemonSnapshot() ([]monitor.TunnelMetrics, *apiSecurityStatus, error) {
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	base := dashboardURL()

	tResp, err := client.Get(base + "/api/v1/tunnels")
	if err != nil {
		return nil, nil, err
	}
	defer tResp.Body.Close()
	if tResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("dashboard unavailable")
	}

	var tunnels []monitor.TunnelMetrics
	if err := json.NewDecoder(tResp.Body).Decode(&tunnels); err != nil {
		return nil, nil, err
	}
	sort.Slice(tunnels, func(i, j int) bool {
		return tunnels[i].Name < tunnels[j].Name
	})

	sResp, err := client.Get(base + "/api/v1/security/status")
	if err != nil {
		return nil, nil, err
	}
	defer sResp.Body.Close()
	if sResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("security status unavailable")
	}

	sec := &apiSecurityStatus{}
	if err := json.NewDecoder(sResp.Body).Decode(sec); err != nil {
		return nil, nil, err
	}

	return tunnels, sec, nil
}

func printStatus() {
	const (
		nameW   = 18
		statusW = 14
		ipW     = 18
	)

	sep := styleDim.Render(strings.Repeat("─", nameW+statusW+ipW+14))
	fmt.Printf("  %s %s %s %s\n",
		styleStatusHdr.Render(pad("PROFILE", nameW)),
		styleStatusHdr.Render(pad("STATUS", statusW)),
		styleStatusHdr.Render(pad("IP", ipW)),
		styleStatusHdr.Render("UPTIME"))
	fmt.Println("  " + sep)

	if tunnels, sec, err := fetchDaemonSnapshot(); err == nil && len(tunnels) > 0 {
		for _, t := range tunnels {
			status := t.Status.Normalize()
			ip := styleDim.Render("—")
			uptime := styleDim.Render("—")
			if t.AssignedIP != "" {
				ip = styleStatusIP.Render(t.AssignedIP)
			}
			if !t.ConnectedAt.IsZero() {
				uptime = styleStatusTime.Render(time.Since(t.ConnectedAt).Round(time.Second).String())
			}

			fmt.Printf("  %s %s %s %s %s\n",
				statusDot(status),
				styleStatusName.Render(pad(t.Name, nameW)),
				pad(statusLabel(status), statusW+30),
				pad(ip, ipW+20),
				uptime)
		}
		fmt.Println("  " + sep)
		fmt.Println("  " + securityGlyph(sec.KillSwitchEnabled, sec.KillSwitch) + "  " + styleBright.Render("Kill switch") + "  " + featureStatus(sec.KillSwitchEnabled, sec.KillSwitch))
		fmt.Println("  " + securityGlyph(sec.DNSGuardEnabled, sec.DNSGuard) + "  " + styleBright.Render("DNS Guard") + "  " + featureStatus(sec.DNSGuardEnabled, sec.DNSGuard))
		return
	}

	for name, adapter := range adapters {
		status := adapter.Status()
		ip := styleDim.Render("—")
		uptime := styleDim.Render("—")

		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			if info.AssignedIP != nil {
				ip = styleStatusIP.Render(info.AssignedIP.String())
			}
			if !info.ConnectedAt.IsZero() {
				uptime = styleStatusTime.Render(time.Since(info.ConnectedAt).Round(time.Second).String())
			}
		}

		fmt.Printf("  %s %s %s %s %s\n",
			statusDot(status),
			styleStatusName.Render(pad(name, nameW)),
			pad(statusLabel(status), statusW+30), // +30 for ANSI bytes
			pad(ip, ipW+20),
			uptime)
	}

	fmt.Println("  " + sep)

	if ks != nil {
		if ks.IsEnabled() {
			fmt.Println("  " + styleStatusUp.Render("⬡") + "  " + styleBright.Render("Kill switch") + "  " + styleStatusUp.Render("ACTIVE"))
		} else {
			fmt.Println("  " + styleStatusDown.Render("⬡") + "  " + styleDim.Render("Kill switch  off"))
		}
	}

	if dnsMgr != nil {
		if dnsMgr.IsActive() {
			fmt.Println("  " + styleStatusUp.Render("⬡") + "  " + styleBright.Render("DNS Guard") + "  " + styleStatusUp.Render("ACTIVE"))
		} else {
			fmt.Println("  " + styleStatusDown.Render("⬡") + "  " + styleDim.Render("DNS Guard  off"))
		}
	}
}

func ternaryStatus(active bool) string {
	if active {
		return styleStatusUp.Render("ACTIVE")
	}
	return styleStatusDown.Render("off")
}

func featureStatus(enabled, active bool) string {
	if !enabled {
		return styleDim.Render("disabled")
	}
	if active {
		return styleStatusUp.Render("ACTIVE")
	}
	return styleStatusDown.Render("idle")
}

func securityGlyph(enabled, active bool) string {
	if !enabled {
		return styleDim.Render("◇")
	}
	if active {
		return styleStatusUp.Render("⬡")
	}
	return styleStatusDown.Render("⬡")
}

func runStatusWatch() error {
	ctx := contextWithSignal()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	clearScreen()
	PrintHeader(version)
	printStatus()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			clearScreen()
			PrintHeader(version)
			fmt.Println("  " + styleDim.Render("Updated "+time.Now().Format("15:04:05")+"  ·  Ctrl+C to exit"))
			fmt.Println()
			printStatus()
		}
	}
}

func clearScreen() {
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
			fmt.Println("  " + styleDim.Render("No routes managed by kongtrol."))
			return nil
		}
		const (
			destW  = 24
			gwW    = 18
			ifaceW = 14
		)
		sep := styleDim.Render("  " + strings.Repeat("─", destW+gwW+ifaceW+10))
		fmt.Printf("  %s %s %s %s\n",
			styleMapHdr.Render(pad("DESTINATION", destW)),
			styleMapHdr.Render(pad("GATEWAY", gwW)),
			styleMapHdr.Render(pad("IFACE", ifaceW)),
			styleMapHdr.Render("METRIC"))
		fmt.Println(sep)
		for _, r := range routes {
			gw := styleDim.Render("—")
			if r.Gateway != nil {
				gw = styleStatusIP.Render(r.Gateway.String())
			}
			fmt.Printf("  %s %s %s %s\n",
				styleMapName.Render(pad(r.Destination.String(), destW)),
				pad(gw, gwW+20),
				styleMapVia.Render(pad(r.Interface, ifaceW)),
				styleStatusTime.Render(fmt.Sprintf("%d", r.Metric)))
		}
		fmt.Println(sep)
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
		spin := newSpinner("Running leak test")
		spin.Start()
		result := leak.CheckNow()
		spin.Stop()
		if result.HasLeak {
			fmt.Println(tuiErr(paintBold(cBright, "Leak detected") + "  " + paint(cDim, result.Reason)))
			return fmt.Errorf("leak detected")
		}
		fmt.Println(tuiOK(paintBold(cBright, "No leak detected") +
			"  " + paint(cDim, "Public IP: ") + styleStatusIP.Render(result.PublicIP)))
		return nil
	},
}

// ── map ──────────────────────────────────────────────────────────────────────

var mapCmd = &cobra.Command{
	Use:   "map [target|app:<exe>]",
	Short: "Show traffic routing map — which VPN handles each destination",
	Long:  "Display all policy rules and their resolved IPs. Optionally query a specific IP/domain or app:<executable>.",
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

var (
	styleMapHdr      = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	styleMapName     = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleMapDomain   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleMapIP       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleMapVia      = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	styleMapResolved = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
)

func printResolve(target string) {
	resolve := func(vpnName string, matched bool) {
		if matched {
			fmt.Printf("  %s  %s  →  %s\n",
				styleStatusUp.Render("●"),
				styleMapName.Render(target),
				styleMapVia.Render(vpnName))
		} else {
			fmt.Printf("  %s  %s  →  %s\n",
				styleDim.Render("○"),
				styleBright.Render(target),
				styleDim.Render("default route (no matching policy)"))
		}
	}

	if ip := net.ParseIP(target); ip != nil {
		vpnName, matched := engine.ResolveIP(ip)
		resolve(vpnName, matched)
	} else if strings.HasPrefix(strings.ToLower(target), "app:") {
		app := strings.TrimSpace(target[4:])
		vpnName, matched := engine.ResolveApp(app)
		resolve(vpnName, matched)
	} else {
		vpnName, matched := engine.ResolveDomain(target)
		resolve(vpnName, matched)
	}
}

func printTrafficMap() {
	rules := engine.Rules()
	if len(rules) == 0 {
		fmt.Println("  " + styleDim.Render("No policies configured."))
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
	if matchW > 40 {
		matchW = 40
	}

	sep := styleDim.Render("  " + strings.Repeat("─", nameW+matchW+viaW+12))
	fmt.Printf("  %s %s %s %s\n",
		styleMapHdr.Render(pad("POLICY", nameW)),
		styleMapHdr.Render(pad("MATCH", matchW)),
		styleMapHdr.Render(pad("VIA", viaW)),
		styleMapHdr.Render("RESOLVED"))
	fmt.Println(sep)

	for _, r := range rules {
		match := summarizeMatch(&r)
		if len(match) > matchW {
			match = match[:matchW-1] + "…"
		}

		resolved := styleDim.Render("—")
		if resolvedByProfile != nil {
			if n, ok := resolvedByProfile[r.Via]; ok && n > 0 {
				resolved = styleMapResolved.Render(fmt.Sprintf("%d IPs", n))
			}
		}

		// Color match column: domains=blue, IPs=orange, mixed=plain
		matchPadded := pad(match, matchW)
		var matchColored string
		if len(r.Match.Domains) > 0 && len(r.Match.IPRanges) == 0 {
			matchColored = styleMapDomain.Render(matchPadded)
		} else if len(r.Match.IPRanges) > 0 && len(r.Match.Domains) == 0 {
			matchColored = styleMapIP.Render(matchPadded)
		} else {
			matchColored = matchPadded
		}

		fmt.Printf("  %s %s %s %s\n",
			styleMapName.Render(pad(r.Name, nameW)),
			matchColored,
			styleMapVia.Render(pad(r.Via, viaW)),
			resolved)
	}
	fmt.Println(sep)
}

// pad right-pads s with spaces to width w (display-safe, no ANSI).
func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func summarizeMatch(r *policy.Rule) string {
	var parts []string
	for _, a := range r.Match.Apps {
		parts = append(parts, "app:"+a)
	}
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
		if _, _, err := fetchDaemonSnapshot(); err == nil {
			addr := dashboardURL()
			fmt.Println(tuiOK(styleBright.Render("Dashboard running") + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
			fmt.Println(paint(cDim, "  Opening browser…"))
			openBrowser(addr)
			return nil
		}

		srv := buildAPIServer()
		if err := srv.Start(); err != nil {
			return fmt.Errorf("dashboard: %w", err)
		}
		addr := dashboardURL()
		fmt.Println(tuiOK(styleBright.Render("Dashboard running") + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
		fmt.Println(paint(cDim, "  Opening browser…"))
		openBrowser(addr)
		fmt.Println(paint(cDim, "  Ctrl+C to stop"))
		fmt.Println()

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
		spin := newSpinner("Validating config")
		spin.Start()
		_, err := config.Load(cfgPath)
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(tuiOK(styleBright.Render("Config is valid")))
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

		// YAML syntax helpers — only colorize on interactive terminals.
		yKey := func(s string) string { return styleMapVia.Render(s) }
		yStr := func(s string) string { return styleStatusIP.Render(s) }
		yNum := func(s string) string { return styleStatusTime.Render(s) }
		yComment := func(s string) string { return styleDim.Render(s) }
		ySection := func(s string) string { return styleGold.Render(s) }

		p := func(indent, key, val string) {
			fmt.Printf("%s%s: %s\n", indent, yKey(key), yStr(val))
		}
		pNum := func(indent, key string, val int) {
			fmt.Printf("%s%s: %s\n", indent, yKey(key), yNum(fmt.Sprintf("%d", val)))
		}
		pComment := func(indent, key, val, comment string) {
			fmt.Printf("%s%s: %s  %s\n", indent, yKey(key), yStr(val), yComment("# "+comment))
		}

		fmt.Println(yComment("# Kongtrol config template — generated by 'kongtrol export'"))
		fmt.Println(yComment("# Passwords and keys are redacted. Run 'kongtrol init' on the target machine"))
		fmt.Println(yComment("# to store credentials in the OS keychain."))
		fmt.Println()
		fmt.Println(ySection("vpns") + yKey(":"))
		for name, v := range cfg.VPNs {
			fmt.Printf("  %s:\n", styleBright.Render(name))
			p("    ", "type", v.Type)
			if v.Version != "" {
				p("    ", "version", v.Version)
			}
			if v.Host != "" {
				p("    ", "host", v.Host)
			}
			if v.Port != 0 {
				pNum("    ", "port", v.Port)
			}
			if v.TunnelName != "" {
				p("    ", "tunnel_name", v.TunnelName)
			}
			if v.ConfigFile != "" {
				p("    ", "config", v.ConfigFile)
			}
			if v.Server != "" {
				p("    ", "server", v.Server)
			}
			if v.Protocol != "" {
				p("    ", "protocol", v.Protocol)
			}
			if v.Priority != 0 {
				pNum("    ", "priority", v.Priority)
			}
			fmt.Printf("    %s:\n", yKey("auth"))
			p("      ", "method", v.Auth.Method)
			if v.Auth.Cert != "" {
				p("      ", "cert", v.Auth.Cert)
			}
			if v.Auth.Key != "" {
				p("      ", "key", v.Auth.Key)
			}
			if v.Auth.Username != "" {
				p("      ", "username", v.Auth.Username)
			}
			if v.Auth.PasswordKeychain != "" {
				pComment("      ", "password_keychain", v.Auth.PasswordKeychain, "store via: kongtrol init")
			}
			if v.Auth.UsernameKeychain != "" {
				pComment("      ", "username_keychain", v.Auth.UsernameKeychain, "store via: kongtrol init")
			}
		}
		if len(cfg.Policies) > 0 {
			fmt.Println()
			fmt.Println(ySection("policies") + yKey(":"))
			for _, pol := range cfg.Policies {
				fmt.Printf("  - %s: %s\n", yKey("name"), yStr(pol.Name))
				if len(pol.Match.IPRanges) > 0 {
					fmt.Printf("    %s:\n      %s: %s\n", yKey("match"), yKey("ip_ranges"), yNum(fmt.Sprintf("%v", pol.Match.IPRanges)))
				}
				if len(pol.Match.Domains) > 0 {
					fmt.Printf("    %s:\n      %s: %s\n", yKey("match"), yKey("domains"), yNum(fmt.Sprintf("%v", pol.Match.Domains)))
				}
				if len(pol.Match.Apps) > 0 {
					fmt.Printf("    %s:\n      %s: %s\n", yKey("match"), yKey("apps"), yNum(fmt.Sprintf("%v", pol.Match.Apps)))
				}
				fmt.Printf("    %s: %s\n", yKey("via"), yStr(pol.Via))
			}
		}
		if len(cfg.Groups) > 0 {
			fmt.Println()
			fmt.Println(ySection("groups") + yKey(":"))
			for name, g := range cfg.Groups {
				fmt.Printf("  %s:\n    %s: %s\n", styleBright.Render(name), yKey("profiles"), yNum(fmt.Sprintf("%v", g.Profiles)))
			}
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
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
	activeCfgPath, err = resolveConfigPath(cfgPath)
	if err != nil {
		return err
	}
	cfg, err = config.Load(activeCfgPath)
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
		Extra: map[string]string{
			"protocol":            vpnCfg.Protocol,
			"allow_insecure_cert": strconv.FormatBool(vpnCfg.AllowInsecureCert),
		},
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

	if adapter.Status().Normalize() != vpn.StatusConnected {
		if err := adapter.Connect(ctx, aCfg); err != nil {
			return err
		}
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
		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			dnsServers := info.DNS
			// Use fallback DNS if tunnel didn't push any servers.
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
		cfg.Security.KillSwitch.Enabled,
		leak,
		engine,
		policyResolver,
		activeCfgPath,
		func(newCfg *config.Config, newEngine *policy.Engine) {
			cfg = newCfg
			engine = newEngine
		},
		dnsMgr,
		cfg.Security.DNSGuard.Enabled,
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
	return "", fmt.Errorf("config: no config file found; run 'kongtrol init' to create one")
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
		fmt.Fprintf(os.Stderr, "warn: cannot stop daemon process %d: %v\n", pid, err)
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
