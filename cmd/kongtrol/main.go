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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"go.uber.org/zap"

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
// Falls back to a local dev marker when built without ldflags.
var version = "v1.0.1-dev"

var (
	cfgPath             string
	activeCfgPath       string
	cfg                 *config.Config
	adapters            map[string]vpn.VPNAdapter
	routeMgr            routing.RouteManager
	engine              *policy.Engine
	ks                  security.KillSwitch
	killSwitchSvc       *app.KillSwitchService
	profileSvc          atomic.Pointer[app.ProfileService]
	leak                *security.LeakTester
	audit               *security.AuditLogger
	col                 *monitor.Collector
	watchdog            *monitor.Watchdog
	scheduler           *monitor.Scheduler
	dnsMgr              *monitor.DNSManager
	policyResolver      *monitor.PolicyResolver
	splitDNSMgr         *monitor.SplitDNSManager
	alertBell           bool
	apiToken            string
	sessionGreetingLine string
	sessionLastUseLine  string
)

var rootCmd = &cobra.Command{
	Use:   "kongtrol",
	Short: "Multi-VPN orchestration — route traffic, enforce security, monitor tunnels",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputPlain {
			_ = os.Setenv("NO_COLOR", "1")
		}
		if cmd.Name() != "init" {
			sessionGreetingLine, sessionLastUseLine = prepareSessionBanner(time.Now(), cmd)
		}
		// init shows its own animated logo — skip compact header and session banner there.
		if cmd.Name() != "init" && !outputJSON && !outputQuiet {
			if !(cmd.Name() == "status" && statusWatch) {
				if cmd.Name() != "version" {
					PrintHeader(version)
				}
				printSessionBanner()
			}
		}
		// Skip config load for commands that self-handle config discovery.
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "doctor" {
			return nil
		}
		return loadConfig()
	},
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SuggestionsMinimumDistance = 2
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%s: %w", ct("cli.flag.error"), err)
	})
	rootCmd.Short = ct("cli.root.short")
	rootCmd.Long = ct("cli.root.long")
	rootCmd.Example = ct("cli.root.examples")

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", ct("cli.flag.config"))
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, ct("cli.flag.json"))
	rootCmd.PersistentFlags().BoolVarP(&outputQuiet, "quiet", "q", false, ct("cli.flag.quiet"))
	rootCmd.PersistentFlags().BoolVar(&outputPlain, "plain", false, ct("cli.flag.plain"))
	rootCmd.PersistentFlags().BoolVar(&alertBell, "alert-bell", false, ct("cli.flag.alert_bell"))
	upCmd.Example = ct("cli.up.examples")
	downCmd.Example = ct("cli.down.examples")
	statusCmd.Example = ct("cli.status.examples")
	routesListCmd.Example = ct("cli.routes.list.examples")
	mapCmd.Example = ct("cli.map.examples")
	dashboardCmd.Example = ct("cli.dashboard.examples")
	configValidateCmd.Example = ct("cli.config.validate.examples")
	upCmd.Short = ct("cli.up.short")
	downCmd.Short = ct("cli.down.short")
	statusCmd.Short = ct("cli.status.short")
	routesCmd.Short = ct("cli.routes.short")
	routesListCmd.Short = ct("cli.routes.list.short")
	checkCmd.Short = ct("cli.check.short")
	mapCmd.Short = ct("cli.map.short")
	dashboardCmd.Short = ct("cli.dashboard.short")
	auditCmd.Short = ct("cli.audit.short")
	configCmd.Short = ct("cli.config.short")
	configValidateCmd.Short = ct("cli.config.validate.short")
	exportCmd.Short = ct("cli.export.short")
	versionCmd.Short = ct("cli.version.short")
	configFavCmd.Short = ct("cli.favorites.short")
	configFavListCmd.Short = ct("cli.favorites.list.short")
	configFavAddCmd.Short = ct("cli.favorites.add.short")
	configFavRemoveCmd.Short = ct("cli.favorites.remove.short")
	configDefaultsCmd.Short = ct("cli.defaults.short")
	configDefaultsShowCmd.Short = ct("cli.defaults.show.short")
	configDefaultsSetGroupCmd.Short = ct("cli.defaults.set_group.short")
	configLangCmd.Short = ct("cli.lang.short")
	configDashboardCmd.Short = ct("cli.config.dashboard.short")
	configDashboardShowCmd.Short = ct("cli.config.dashboard.show.short")
	configDashboardSetPortCmd.Short = ct("cli.config.dashboard.set_port.short")
	configDashboardSetBindCmd.Short = ct("cli.config.dashboard.set_bind.short")
	configDashboardResetCmd.Short = ct("cli.config.dashboard.reset.short")

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
var upDryRun bool
var upFavorites bool

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
			targets, err = resolveUpProfiles(args, upGroup, upFavorites)
			if err != nil {
				return err
			}
		}
		if upDryRun {
			return runUpDryRun(targets)
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
					fmt.Println(tuiWarn(ct("cli.up.cancelled")))
					return nil
				}
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
					err = fmt.Errorf("%s", ct("cli.up.timeout"))
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
		fmt.Println(tuiInfo(paint(cDim, cf("cli.up.connected_all", time.Since(startedAll).Round(time.Second)))))

		// Keep kill switch synchronized to active protected profiles.
		_ = applyKillSwitchState()
		if ks != nil {
			defer func() { _ = ks.Disable() }()
		}

		// Start watchdog after all profiles are up.
		if watchdog != nil {
			watchdog.Start(ctx)
			defer watchdog.Stop()
		}
		if scheduler != nil && cfg.Monitor.Scheduler.Enabled {
			scheduler.Start(ctx)
			defer scheduler.Stop()
		}

		// Start background DNS resolver for domain-based split tunneling.
		if policyResolver != nil {
			policyResolver.Start(ctx)
			defer policyResolver.Stop()
		}
		if splitDNSMgr != nil && cfg.Monitor.SplitDNS.Enabled {
			splitDNSMgr.Start(ctx)
			defer splitDNSMgr.Stop()
		}
		startHistoryPersistence(ctx)

		// Start background leak detection.
		if leak != nil {
			leak.Start(ctx, func(result security.LeakResult) {
				if result.HasLeak {
					if col != nil {
						for name, a := range adapters {
							if a.Status().Normalize() == vpn.StatusConnected {
								col.RecordLeak(name)
							}
						}
					}
					emitAlert("ERROR", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
					logAudit("SECURITY", "security.leak", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
				}
			})
			defer leak.Stop()
		}

		// Start the API server / dashboard alongside the daemon.
		var dashURL string
		srv := buildAPIServer(cancelCtx)
		if err := srv.Start(); err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.up.warn.dashboard_server", err)))
		} else {
			dashURL = srv.Addr()
			defer func() {
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutCtx)
			}()
		}

		// Block until signal — show live daemon view on interactive terminals.
		runUpTUI(ctx, cancelCtx, adapters, ks, dnsMgr, dashURL, true)
		return nil
	},
}

func init() {
	upCmd.Flags().BoolVar(&upAll, "all", false, ct("cli.up.flag.all"))
	upCmd.Flags().StringVar(&upGroup, "group", "", ct("cli.up.flag.group"))
	upCmd.Flags().BoolVar(&upDryRun, "dry-run", false, ct("cli.up.dry_run_help"))
	upCmd.Flags().BoolVar(&upFavorites, "fav", false, ct("cli.up.flag.fav"))
}

// ── down ─────────────────────────────────────────────────────────────────────

var downAll bool
var downGroup string

var downCmd = &cobra.Command{
	Use:   "down [profile...]",
	Short: "Disconnect one or more VPN profiles (or a group with --group)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := contextWithSignal()
		disconnectWithSpinner := func(name string) error {
			spin := newSpinner(cf("cli.down.disconnecting", name))
			spin.Start()
			err := disconnectProfile(ctx, name)
			spin.Stop()
			if err != nil {
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error()))
				return err
			}
			fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, ct("cli.down.disconnected"))))
			return nil
		}

		if downAll {
			names := make([]string, 0, len(adapters))
			for name := range adapters {
				names = append(names, name)
			}
			disconnectAllConcurrently(ctx, names, disconnectProfile)
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
	downCmd.Flags().BoolVar(&downAll, "all", false, ct("cli.down.flag.all"))
	downCmd.Flags().StringVar(&downGroup, "group", "", ct("cli.down.flag.group"))
}

// disconnectAllConcurrently disconnects every named profile in parallel
// instead of one at a time — with several active tunnels the sequential
// version added up to real wait time for no benefit, since each profile's
// adapter is independent.
//
// Per-profile live spinners (used by the single/multi-target path above)
// aren't used here: they animate by repeatedly overwriting the current
// terminal line via \r, and multiple concurrent spinners racing on that
// same line would corrupt the display. Instead this shows one aggregate
// spinner while all disconnects run, then prints each result once
// everything has settled. Matches the original --all semantics: every
// profile is attempted and its error (if any) is reported, but a single
// profile failing doesn't stop the others or fail the command.
func disconnectAllConcurrently(ctx context.Context, names []string, disconnect func(context.Context, string) error) {
	if len(names) == 0 {
		return
	}

	agg := newSpinner(cf("cli.down.disconnecting_all", len(names)))
	agg.Start()

	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- result{name: name, err: disconnect(ctx, name)}
		}(name)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	lines := make([]string, 0, len(names))
	for r := range results {
		if r.err != nil {
			lines = append(lines, tuiErr(paintBold(cBright, r.name)+"  "+r.err.Error()))
		} else {
			lines = append(lines, tuiOK(paintBold(cBright, r.name)+"  "+paint(cDim, ct("cli.down.disconnected"))))
		}
	}
	agg.Stop()
	for _, l := range lines {
		fmt.Println(l)
	}
}

// ── status ───────────────────────────────────────────────────────────────────

var statusWatch bool
var statusDashboard bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all VPN tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusDashboard && !statusWatch {
			return fmt.Errorf("%s", ct("cli.status.error.dashboard_requires_watch"))
		}
		if outputJSON {
			report, err := collectStatusReport()
			if err != nil {
				return err
			}
			return emitJSON(report)
		}
		if statusWatch {
			return runStatusWatch()
		}
		printStatus()
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, ct("cli.status.flag.watch"))
	statusCmd.Flags().BoolVar(&statusDashboard, "dashboard", false, ct("cli.status.flag.dashboard"))
}

// Status table styles live in theme.go ("Signal Contour").

const (
	minProfileColW = 12
	minStatusColW  = 12
	minIPColW      = 12
	minUptimeColW  = 10
)

func statusDot(s vpn.Status) string {
	switch s {
	case vpn.StatusConnected:
		return styleStatusUp.Render(sym("●", "*"))
	case vpn.StatusConnecting:
		return styleInfo.Render(sym("◐", "~"))
	case vpn.StatusError:
		return styleStatusErr.Render(sym("✗", "x"))
	default:
		return styleStatusDown.Render(sym("○", "o"))
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

func formatUptime(now, connectedAt time.Time) string {
	if connectedAt.IsZero() {
		return "—"
	}
	d := now.Sub(connectedAt)
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
}

func formatElapsedCompact(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatStatusUptime(now time.Time, status vpn.Status, connectedAt time.Time, h monitor.ProfileHistory) string {
	if status == vpn.StatusConnected {
		since := connectedAt
		if since.IsZero() {
			since = h.LastConnectedAt
		}
		if since.IsZero() {
			return "—"
		}
		return cf("cli.status.uptime.connected", formatElapsedCompact(now.Sub(since)), since.Local().Format("15:04:05"))
	}

	if !h.LastDownAt.IsZero() {
		return cf("cli.status.uptime.disconnected", formatElapsedCompact(now.Sub(h.LastDownAt)), h.LastDownAt.Local().Format("15:04:05"))
	}
	if !h.LastConnectedAt.IsZero() {
		return cf("cli.status.uptime.last_up", h.LastConnectedAt.Local().Format("2006-01-02 15:04:05"))
	}
	return "—"
}

func prepareSessionBanner(now time.Time, cmd *cobra.Command) (greeting, lastUse string) {
	path := sessionStatePath()
	prev, err := loadSessionState(path)
	if err == nil {
		lastAt := prev.LastCommandAt
		if lastAt.IsZero() {
			lastAt = prev.LastLoginAt
		}
		if !lastAt.IsZero() {
			if strings.TrimSpace(prev.LastCommand) != "" {
				lastUse = cf("cli.session.last_use", lastAt.Local().Format("Mon Jan 02 15:04:05 2006"), prev.LastCommand)
			} else {
				lastUse = cf("cli.session.last_use_simple", lastAt.Local().Format("Mon Jan 02 15:04:05 2006"))
			}
		}
	}

	greeting = cf(statusGreetingKeyForHour(now.Hour()), resolveSystemUserName())
	_ = saveSessionState(path, cliSessionState{
		LastCommandAt: now,
		LastCommand:   cmd.CommandPath(),
		LastLoginAt:   now, // legacy compatibility
	})
	return greeting, lastUse
}

func statusGreetingKeyForHour(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "cli.status.greeting.morning"
	case hour >= 12 && hour < 20:
		return "cli.status.greeting.afternoon"
	default:
		return "cli.status.greeting.night"
	}
}

func sortedAdapterNames(m map[string]vpn.VPNAdapter) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type tunnelRow struct {
	Name   string
	Status vpn.Status
	IP     string
	Uptime string
}

type tunnelTableLayout struct {
	ProfileW int
	StatusW  int
	IPW      int
	UptimeW  int
	RuleW    int
}

func terminalWidth() int {
	if v := strings.TrimSpace(os.Getenv("COLUMNS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 60 {
			return n
		}
	}
	return 110
}

func fitCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s + strings.Repeat(" ", width-lipgloss.Width(s))
	}
	if width == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if used+w > width-1 {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String() + "…"
}

func statusText(s vpn.Status) string {
	n := s.Normalize()
	if n == "" {
		n = vpn.StatusDisconnected
	}
	switch n {
	case vpn.StatusConnected:
		return ct("cli.status.value.connected")
	case vpn.StatusConnecting:
		return ct("cli.status.value.connecting")
	case vpn.StatusError:
		return ct("cli.status.value.error")
	default:
		return ct("cli.status.value.disconnected")
	}
}

func statusRawCell(s vpn.Status) string {
	return sym(statusGlyphRaw(s), statusGlyphPlain(s)) + " " + statusText(s)
}

func statusGlyphRaw(s vpn.Status) string {
	switch s.Normalize() {
	case vpn.StatusConnected:
		return "●"
	case vpn.StatusConnecting:
		return "◐"
	case vpn.StatusError:
		return "✗"
	default:
		return "○"
	}
}

func statusGlyphPlain(s vpn.Status) string {
	switch s.Normalize() {
	case vpn.StatusConnected:
		return "*"
	case vpn.StatusConnecting:
		return "~"
	case vpn.StatusError:
		return "x"
	default:
		return "o"
	}
}

func shrinkWidth(current *int, min int, need *int) {
	if *need <= 0 || *current <= min {
		return
	}
	step := *current - min
	if step > *need {
		step = *need
	}
	*current -= step
	*need -= step
}

func computeTunnelTableLayout(rows []tunnelRow, totalWidth int) tunnelTableLayout {
	hProfile := lipgloss.Width(ct("cli.status.col.profile"))
	hStatus := lipgloss.Width(ct("cli.status.col.status"))
	hIP := lipgloss.Width("IP")
	hUptime := lipgloss.Width(ct("cli.status.col.uptime"))

	maxProfile, maxStatus, maxIP, maxUptime := hProfile, hStatus, hIP, hUptime
	for _, r := range rows {
		if w := lipgloss.Width(r.Name); w > maxProfile {
			maxProfile = w
		}
		if w := lipgloss.Width(statusRawCell(r.Status)); w > maxStatus {
			maxStatus = w
		}
		if w := lipgloss.Width(r.IP); w > maxIP {
			maxIP = w
		}
		if w := lipgloss.Width(r.Uptime); w > maxUptime {
			maxUptime = w
		}
	}

	layout := tunnelTableLayout{
		ProfileW: max(minProfileColW, min(28, maxProfile+1)),
		StatusW:  max(minStatusColW, min(18, maxStatus+1)),
		IPW:      max(minIPColW, min(40, maxIP+1)),
		UptimeW:  max(minUptimeColW, min(32, maxUptime+1)),
	}

	available := totalWidth - 2 - 3 // indent + separators
	sum := layout.ProfileW + layout.StatusW + layout.IPW + layout.UptimeW
	if sum > available {
		need := sum - available
		shrinkWidth(&layout.IPW, minIPColW, &need)
		shrinkWidth(&layout.ProfileW, minProfileColW, &need)
		shrinkWidth(&layout.StatusW, minStatusColW, &need)
		shrinkWidth(&layout.UptimeW, minUptimeColW, &need)
	}
	if sum2 := layout.ProfileW + layout.StatusW + layout.IPW + layout.UptimeW; sum2 < available {
		extra := available - sum2
		layout.IPW += extra / 2
		layout.ProfileW += extra - (extra / 2)
	}
	layout.RuleW = layout.ProfileW + layout.StatusW + layout.IPW + layout.UptimeW + 3
	return layout
}

func tableRule(width int) string {
	if width < 1 {
		width = 1
	}
	return styleDim.Render(strings.Repeat("─", width))
}

func renderTunnelHeader(l tunnelTableLayout) string {
	return fmt.Sprintf("  %s %s %s %s",
		styleStatusHdr.Render(fitCell(ct("cli.status.col.profile"), l.ProfileW)),
		styleStatusHdr.Render(fitCell(ct("cli.status.col.status"), l.StatusW)),
		styleStatusHdr.Render(fitCell("IP", l.IPW)),
		styleStatusHdr.Render(fitCell(ct("cli.status.col.uptime"), l.UptimeW)))
}

func renderTunnelRow(l tunnelTableLayout, r tunnelRow) string {
	profile := styleStatusName.Render(fitCell(r.Name, l.ProfileW))
	var statusCell string
	switch r.Status.Normalize() {
	case vpn.StatusConnected:
		statusCell = styleStatusUp.Render(fitCell(statusRawCell(r.Status), l.StatusW))
	case vpn.StatusConnecting:
		statusCell = styleInfo.Render(fitCell(statusRawCell(r.Status), l.StatusW))
	case vpn.StatusError:
		statusCell = styleStatusErr.Render(fitCell(statusRawCell(r.Status), l.StatusW))
	default:
		statusCell = styleStatusDown.Render(fitCell(statusRawCell(r.Status), l.StatusW))
	}
	ip := styleStatusIP.Render(fitCell(r.IP, l.IPW))
	uptime := styleStatusTime.Render(fitCell(r.Uptime, l.UptimeW))
	return fmt.Sprintf("  %s %s %s %s", profile, statusCell, ip, uptime)
}

func statusCounts(rows []tunnelRow) (connected, connecting, errored, disconnected int) {
	for _, r := range rows {
		switch r.Status.Normalize() {
		case vpn.StatusConnected:
			connected++
		case vpn.StatusConnecting:
			connecting++
		case vpn.StatusError:
			errored++
		default:
			disconnected++
		}
	}
	return
}

func renderStatusSummary(rows []tunnelRow) string {
	c, cg, e, d := statusCounts(rows)
	parts := []string{
		styleStatusUp.Render(cf("cli.status.summary.connected", c)),
		styleInfo.Render(cf("cli.status.summary.connecting", cg)),
		styleStatusErr.Render(cf("cli.status.summary.error", e)),
		styleStatusDown.Render(cf("cli.status.summary.disconnected", d)),
	}
	return "  " + strings.Join(parts, styleDim.Render("  ·  "))
}

func renderSecurityLine(ksEnabled, ksActive, dnsEnabled, dnsActive bool) string {
	killState := ct("cli.status.security.idle_caps")
	killStateStyle := styleStatusDown
	if !ksEnabled {
		killState = ct("cli.status.security.disabled_caps")
		killStateStyle = styleDim
	} else if ksActive {
		killState = ct("cli.status.security.armed")
		killStateStyle = styleStatusUp
	}
	dnsState := ct("cli.status.security.idle_caps")
	dnsStateStyle := styleStatusDown
	if !dnsEnabled {
		dnsState = ct("cli.status.security.disabled_caps")
		dnsStateStyle = styleDim
	} else if dnsActive {
		dnsState = ct("cli.status.security.active_caps")
		dnsStateStyle = styleStatusUp
	}
	killIcon := styleDim.Render(sym("○", "o"))
	if ksEnabled {
		killIcon = killStateStyle.Render(sym("●", "*"))
	}
	return killIcon + " " +
		styleBright.Render(ct("cli.status.kill_switch")) + " " + killStateStyle.Render(killState) +
		styleDim.Render("  ·  ") +
		styleBright.Render(ct("cli.status.dns_guard")) + " " + dnsStateStyle.Render(dnsState)
}

func renderLeakCheckLine(configured, available bool, check *apiLeakCheck) string {
	state := ct("cli.status.security.disabled_caps")
	stateStyle := styleDim
	if configured && !available {
		state = ct("cli.status.security.unavailable_caps")
		stateStyle = styleStatusErr
	}
	if available {
		state = ct("cli.status.security.pending_caps")
		stateStyle = styleStatusDown
	}
	if check != nil {
		switch check.State {
		case "clean":
			state = ct("cli.status.security.clean_caps")
			stateStyle = styleStatusUp
		case "leak":
			state = ct("cli.status.security.leak_caps")
			stateStyle = styleStatusErr
		case "error":
			state = ct("cli.status.security.error_caps")
			stateStyle = styleStatusErr
		}
	}
	return styleBright.Render(ct("cli.status.leak_check")) + " " + stateStyle.Render(state)
}

func renderWatchdogLine() string {
	return styleInfo.Render(sym("●", "~")) + " " + styleBright.Render(cf("cli.status.watchdog.line", "5s", "2s", "5m"))
}

type apiSecurityStatus struct {
	KillSwitch              bool          `json:"kill_switch"`
	KillSwitchEnabled       bool          `json:"kill_switch_enabled"`
	DNSGuard                bool          `json:"dns_guard"`
	DNSGuardEnabled         bool          `json:"dns_guard_enabled"`
	LeakDetectionEnabled    bool          `json:"leak_detection_enabled"`
	LeakDetectionConfigured bool          `json:"leak_detection_configured"`
	LeakCheckAvailable      bool          `json:"leak_check_available"`
	LeakCheck               *apiLeakCheck `json:"leak_check,omitempty"`
}

type apiLeakCheck struct {
	State     string    `json:"state"`
	HasLeak   bool      `json:"has_leak"`
	PublicIP  string    `json:"public_ip,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

func leakResultToAPI(result *security.LeakResult) *apiLeakCheck {
	state := "clean"
	if result.HasLeak {
		state = "leak"
	} else if result.PublicIP == "" && result.Reason != "" {
		state = "error"
	}
	return &apiLeakCheck{
		State:     state,
		HasLeak:   result.HasLeak,
		PublicIP:  result.PublicIP,
		Reason:    result.Reason,
		CheckedAt: result.CheckedAt,
	}
}

type statusTunnelJSON struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	AssignedIP  string    `json:"assigned_ip,omitempty"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
	UptimeSec   int64     `json:"uptime_sec"`
}

type statusSecurityJSON struct {
	KillSwitchEnabled       bool          `json:"kill_switch_enabled"`
	KillSwitchActive        bool          `json:"kill_switch_active"`
	DNSGuardEnabled         bool          `json:"dns_guard_enabled"`
	DNSGuardActive          bool          `json:"dns_guard_active"`
	LeakDetectionConfigured bool          `json:"leak_detection_configured"`
	LeakCheckAvailable      bool          `json:"leak_check_available"`
	LeakCheck               *apiLeakCheck `json:"leak_check,omitempty"`
}

type statusReportJSON struct {
	GeneratedAt time.Time                         `json:"generated_at"`
	Source      string                            `json:"source"`
	Tunnels     []statusTunnelJSON                `json:"tunnels"`
	Security    statusSecurityJSON                `json:"security"`
	User        string                            `json:"user,omitempty"`
	LastUse     string                            `json:"last_use,omitempty"`
	History     map[string]monitor.ProfileHistory `json:"history,omitempty"`
}

func dashboardURL() string {
	return fmt.Sprintf("http://%s:%d", cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port)
}

func collectStatusReport() (statusReportJSON, error) {
	now := time.Now()
	report := statusReportJSON{
		GeneratedAt: now,
		Source:      "local",
		Tunnels:     make([]statusTunnelJSON, 0, len(adapters)),
		Security: statusSecurityJSON{
			KillSwitchEnabled:       cfg.Security.KillSwitch.Enabled,
			DNSGuardEnabled:         cfg.Security.DNSGuard.Enabled,
			LeakDetectionConfigured: cfg.Security.LeakDetection.Enabled,
			LeakCheckAvailable:      leak != nil,
		},
	}
	report.User = resolveSystemUserName()
	if sessionLastUseLine != "" {
		report.LastUse = sessionLastUseLine
	}

	if tunnels, sec, err := fetchDaemonSnapshot(); err == nil {
		report.Source = "daemon"
		report.Security.KillSwitchActive = sec.KillSwitch
		report.Security.KillSwitchEnabled = sec.KillSwitchEnabled
		report.Security.DNSGuardActive = sec.DNSGuard
		report.Security.DNSGuardEnabled = sec.DNSGuardEnabled
		report.Security.LeakDetectionConfigured = sec.LeakDetectionConfigured
		report.Security.LeakCheckAvailable = sec.LeakCheckAvailable
		report.Security.LeakCheck = sec.LeakCheck
		for _, t := range tunnels {
			item := statusTunnelJSON{
				Name:       t.Name,
				Status:     string(t.Status.Normalize()),
				AssignedIP: t.AssignedIP,
			}
			if !t.ConnectedAt.IsZero() {
				item.ConnectedAt = t.ConnectedAt
				item.UptimeSec = int64(time.Since(t.ConnectedAt).Seconds())
				if item.UptimeSec < 0 {
					item.UptimeSec = 0
				}
			}
			report.Tunnels = append(report.Tunnels, item)
		}
		if h, err := fetchDaemonHistory(); err == nil {
			report.History = h
		}
		return report, nil
	}

	if ks != nil {
		report.Security.KillSwitchActive = ks.IsEnabled()
	}
	if dnsMgr != nil {
		report.Security.DNSGuardActive = dnsMgr.IsActive()
	}
	if leak != nil {
		if result := leak.LastResult(); result != nil {
			report.Security.LeakCheck = leakResultToAPI(result)
		}
	}
	for _, name := range sortedAdapterNames(adapters) {
		adapter := adapters[name]
		status := adapter.Status().Normalize()
		item := statusTunnelJSON{
			Name:   name,
			Status: string(status),
		}
		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			if info.AssignedIP != nil {
				item.AssignedIP = info.AssignedIP.String()
			}
			if !info.ConnectedAt.IsZero() {
				item.ConnectedAt = info.ConnectedAt
				item.UptimeSec = int64(time.Since(info.ConnectedAt).Seconds())
				if item.UptimeSec < 0 {
					item.UptimeSec = 0
				}
			}
		}
		report.Tunnels = append(report.Tunnels, item)
	}
	if col != nil {
		report.History = col.HistorySnapshot()
	}

	return report, nil
}

func fetchDaemonHistory() (map[string]monitor.ProfileHistory, error) {
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	req, err := daemonRequest(http.MethodGet, dashboardURL()+"/api/v1/metrics/history", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history unavailable")
	}
	out := map[string]monitor.ProfileHistory{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func fetchDaemonSnapshot() ([]monitor.TunnelMetrics, *apiSecurityStatus, error) {
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	base := dashboardURL()

	tReq, err := daemonRequest(http.MethodGet, base+"/api/v1/tunnels", nil)
	if err != nil {
		return nil, nil, err
	}
	tResp, err := client.Do(tReq)
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

	sReq, err := daemonRequest(http.MethodGet, base+"/api/v1/security/status", nil)
	if err != nil {
		return nil, nil, err
	}
	sResp, err := client.Do(sReq)
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
	now := time.Now()
	width := terminalWidth()

	tunnels, sec, daemonErr := fetchDaemonSnapshot()
	if daemonErr == nil {
		history, _ := fetchDaemonHistory()
		seen := make(map[string]bool, len(tunnels))
		rows := make([]tunnelRow, 0, len(tunnels)+len(adapters))
		for _, t := range tunnels {
			status := t.Status.Normalize()
			ip := "—"
			if t.AssignedIP != "" {
				ip = t.AssignedIP
			}
			uptime := formatStatusUptime(now, status, t.ConnectedAt, history[t.Name])
			rows = append(rows, tunnelRow{
				Name:   t.Name,
				Status: status,
				IP:     ip,
				Uptime: uptime,
			})
			seen[t.Name] = true
		}

		// Ensure all configured profiles are shown even when daemon omits inactive tunnels.
		for _, name := range sortedAdapterNames(adapters) {
			if seen[name] {
				continue
			}
			rows = append(rows, tunnelRow{
				Name:   name,
				Status: vpn.StatusDisconnected,
				IP:     "—",
				Uptime: formatStatusUptime(now, vpn.StatusDisconnected, time.Time{}, history[name]),
			})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

		layout := computeTunnelTableLayout(rows, width)
		sep := tableRule(layout.RuleW)
		fmt.Println(renderStatusSummary(rows))
		fmt.Println("  " + renderSecurityLine(sec.KillSwitchEnabled, sec.KillSwitch, sec.DNSGuardEnabled, sec.DNSGuard))
		fmt.Println("  " + renderLeakCheckLine(sec.LeakDetectionConfigured, sec.LeakCheckAvailable, sec.LeakCheck))
		fmt.Println("  " + renderWatchdogLine())
		fmt.Println()
		fmt.Println(renderTunnelHeader(layout))
		fmt.Println("  " + sep)
		for _, r := range rows {
			fmt.Println(renderTunnelRow(layout, r))
		}
		fmt.Println("  " + sep)
		return
	}

	rows := make([]tunnelRow, 0, len(adapters))
	history := map[string]monitor.ProfileHistory{}
	if col != nil {
		history = col.HistorySnapshot()
	}
	for _, name := range sortedAdapterNames(adapters) {
		adapter := adapters[name]
		status := adapter.Status().Normalize()
		ip := "—"
		var connectedAt time.Time
		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			if info.AssignedIP != nil {
				ip = info.AssignedIP.String()
			}
			connectedAt = info.ConnectedAt
		}
		rows = append(rows, tunnelRow{
			Name:   name,
			Status: status,
			IP:     ip,
			Uptime: formatStatusUptime(now, status, connectedAt, history[name]),
		})
	}

	ksEnabled, ksActive := false, false
	dnsEnabled, dnsActive := false, false
	if cfg != nil {
		ksEnabled = cfg.Security.KillSwitch.Enabled
		dnsEnabled = cfg.Security.DNSGuard.Enabled
	}
	if ks != nil {
		ksActive = ks.IsEnabled()
	}
	if dnsMgr != nil {
		dnsActive = dnsMgr.IsActive()
	}

	layout := computeTunnelTableLayout(rows, width)
	sep := tableRule(layout.RuleW)
	fmt.Println(renderStatusSummary(rows))
	fmt.Println("  " + renderSecurityLine(ksEnabled, ksActive, dnsEnabled, dnsActive))
	var leakCheck *apiLeakCheck
	if leak != nil {
		if result := leak.LastResult(); result != nil {
			leakCheck = leakResultToAPI(result)
		}
	}
	fmt.Println("  " + renderLeakCheckLine(cfg != nil && cfg.Security.LeakDetection.Enabled, leak != nil, leakCheck))
	fmt.Println("  " + renderWatchdogLine())
	fmt.Println()
	fmt.Println(renderTunnelHeader(layout))
	fmt.Println("  " + sep)
	for _, r := range rows {
		fmt.Println(renderTunnelRow(layout, r))
	}
	fmt.Println("  " + sep)
}

func ternaryStatus(active bool) string {
	if active {
		return styleStatusUp.Render(ct("cli.status.security.active"))
	}
	return styleStatusDown.Render(ct("cli.status.off"))
}

func featureStatus(enabled, active bool) string {
	if !enabled {
		return styleDim.Render(ct("cli.status.security.disabled"))
	}
	if active {
		return styleStatusUp.Render(ct("cli.status.security.active"))
	}
	return styleStatusDown.Render(ct("cli.status.security.idle"))
}

func securityGlyph(enabled, active bool) string {
	if !enabled {
		return styleDim.Render(sym("·", "-"))
	}
	if active {
		return styleStatusUp.Render(sym("●", "*"))
	}
	return styleStatusDown.Render(sym("○", "o"))
}

// runStatusWatch shows the same live Bubble Tea daemon view as `up` — a
// viewer into the currently running tunnels (no daemon lifecycle is started
// here; `up` owns that in its own process). If `up` is already running,
// runUpTUI detects its API server and proxies connect/disconnect/reconnect
// actions to it; otherwise those actions are disabled since this process has
// no coordinated daemon to hand the resulting tunnel off to.
func runStatusWatch() error {
	signalCtx := contextWithSignal()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	var dashURL string
	if statusDashboard {
		base := daemonAPIBase()
		if probeDaemonAPI(base) {
			// A daemon's dashboard is already serving on this address —
			// reuse it instead of trying (and failing) to bind the same port.
			dashURL = base
		} else {
			srv := buildAPIServer(cancel)
			if err := srv.Start(); err != nil {
				return err
			}
			dashURL = srv.Addr()
			defer func() {
				shutCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
				defer c()
				_ = srv.Shutdown(shutCtx)
			}()
		}
	}

	runUpTUI(ctx, cancel, adapters, ks, dnsMgr, dashURL, false)
	return nil
}

func printSessionBanner() {
	if sessionGreetingLine != "" {
		fmt.Println("  " + styleDim.Render(sessionGreetingLine))
	}
	if sessionLastUseLine != "" {
		fmt.Println("  " + styleDim.Render(sessionLastUseLine))
	}
	if sessionGreetingLine != "" || sessionLastUseLine != "" {
		fmt.Println()
	}
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
		if outputJSON {
			return emitJSON(struct {
				Count  int             `json:"count"`
				Routes []routing.Route `json:"routes"`
			}{
				Count:  len(routes),
				Routes: routes,
			})
		}
		if len(routes) == 0 {
			fmt.Println("  " + styleDim.Render(ct("cli.routes.none")))
			return nil
		}
		destW, gwW, ifaceW, metricW := 24, 20, 14, 6
		for _, r := range routes {
			destW = max(destW, min(42, lipgloss.Width(r.Destination.String())+1))
			if r.Gateway != nil {
				gwW = max(gwW, min(40, lipgloss.Width(r.Gateway.String())+1))
			}
			ifaceW = max(ifaceW, min(24, lipgloss.Width(r.Interface)+1))
		}
		available := terminalWidth() - 2 - 3 // indent + spaces
		sum := destW + gwW + ifaceW + metricW
		if sum > available {
			need := sum - available
			shrinkWidth(&destW, 16, &need)
			shrinkWidth(&gwW, 14, &need)
			shrinkWidth(&ifaceW, 10, &need)
		}
		ruleW := destW + gwW + ifaceW + metricW + 3
		sep := styleDim.Render("  " + strings.Repeat("─", ruleW))
		fmt.Printf("  %s %s %s %s\n",
			styleMapHdr.Render(fitCell(ct("cli.routes.col.dest"), destW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.gateway"), gwW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.iface"), ifaceW)),
			styleMapHdr.Render(fitCell(ct("cli.routes.col.metric"), metricW)))
		fmt.Println(sep)
		for _, r := range routes {
			gw := "—"
			if r.Gateway != nil {
				gw = r.Gateway.String()
			}
			fmt.Printf("  %s %s %s %s\n",
				styleMapName.Render(fitCell(r.Destination.String(), destW)),
				styleStatusIP.Render(fitCell(gw, gwW)),
				styleMapVia.Render(fitCell(r.Interface, ifaceW)),
				styleStatusTime.Render(fitCell(fmt.Sprintf("%d", r.Metric), metricW)))
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
			return fmt.Errorf("%s", ct("cli.check.leak_not_initialized"))
		}
		var spin *spinner
		if !outputJSON {
			spin = newSpinner(ct("cli.check.spinner"))
			spin.Start()
		}
		result := leak.CheckNow()
		if spin != nil {
			spin.Stop()
		}
		if outputJSON {
			if err := emitJSON(struct {
				Leak     bool   `json:"leak"`
				Reason   string `json:"reason,omitempty"`
				PublicIP string `json:"public_ip,omitempty"`
			}{
				Leak:     result.HasLeak,
				Reason:   result.Reason,
				PublicIP: result.PublicIP,
			}); err != nil {
				return err
			}
			if result.HasLeak {
				return fmt.Errorf("%s", ct("cli.check.leak_detected"))
			}
			return nil
		}
		if result.HasLeak {
			fmt.Println(tuiErr(paintBold(cBright, ct("cli.check.leak_detected")) + "  " + paint(cDim, result.Reason)))
			return fmt.Errorf("%s", ct("cli.check.leak_detected"))
		}
		fmt.Println(tuiOK(paintBold(cBright, ct("cli.check.no_leak")) +
			"  " + paint(cDim, ct("cli.check.public_ip")) + styleStatusIP.Render(result.PublicIP)))
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
			return fmt.Errorf("%s", ct("cli.policy.engine_not_loaded"))
		}
		if outputJSON {
			return emitJSON(buildMapReport(args))
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

type mapRuleJSON struct {
	Name        string   `json:"name"`
	Via         string   `json:"via"`
	Apps        []string `json:"apps,omitempty"`
	Domains     []string `json:"domains,omitempty"`
	IPRanges    []string `json:"ip_ranges,omitempty"`
	ResolvedIPs int      `json:"resolved_ips"`
}

type mapReportJSON struct {
	Target *policy.ExplainResult `json:"target,omitempty"`
	Rules  []mapRuleJSON         `json:"rules"`
}

// Map table styles live in theme.go ("Signal Contour").

func printResolve(target string) {
	ex := engine.ExplainTarget(target)
	if ex.Matched {
		fmt.Printf("  %s  %s  →  %s\n",
			styleStatusUp.Render("●"),
			styleMapName.Render(target),
			styleMapVia.Render(ex.Via))
		if ex.RuleName != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.rule_label")) + styleBright.Render(ex.RuleName))
		}
		if ex.Reason != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
		}
	} else {
		fmt.Printf("  %s  %s  →  %s\n",
			styleDim.Render("○"),
			styleBright.Render(target),
			styleDim.Render(ct("cli.policy.default_route")))
		if ex.Reason != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
		}
	}
}

func printTrafficMap() {
	rules := engine.Rules()
	if len(rules) == 0 {
		fmt.Println("  " + styleDim.Render(ct("cli.map.none")))
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

func buildMapReport(args []string) mapReportJSON {
	out := mapReportJSON{Rules: []mapRuleJSON{}}
	if len(args) > 0 {
		ex := engine.ExplainTarget(args[0])
		out.Target = &ex
	}

	rules := engine.Rules()
	if len(rules) == 0 {
		return out
	}

	resolvedByProfile := map[string]int{}
	if policyResolver != nil {
		for _, snap := range policyResolver.Snapshot() {
			resolvedByProfile[snap.Name] = len(snap.ResolvedCIDRs)
		}
	}

	out.Rules = make([]mapRuleJSON, 0, len(rules))
	for _, r := range rules {
		row := mapRuleJSON{
			Name:     r.Name,
			Via:      r.Via,
			Apps:     append([]string(nil), r.Match.Apps...),
			Domains:  append([]string(nil), r.Match.Domains...),
			IPRanges: make([]string, 0, len(r.Match.IPRanges)),
		}
		for _, p := range r.Match.IPRanges {
			row.IPRanges = append(row.IPRanges, p.String())
		}
		row.ResolvedIPs = resolvedByProfile[r.Via]
		out.Rules = append(out.Rules, row)
	}

	return out
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
			fmt.Println(tuiOK(styleBright.Render(ct("cli.dashboard.running")) + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
			fmt.Println(paint(cDim, "  "+ct("cli.dashboard.opening")))
			if err := openBrowser(addr); err != nil {
				fmt.Println(tuiWarn(cf("cli.dashboard.open_failed", err)))
			}
			return nil
		}

		ctx, cancel := context.WithCancel(contextWithSignal())
		defer cancel()

		srv := buildAPIServer(cancel)
		if err := srv.Start(); err != nil {
			return fmt.Errorf("%s", cf("cli.dashboard.error_start", err))
		}
		addr := dashboardURL()
		fmt.Println(tuiOK(styleBright.Render(ct("cli.dashboard.running")) + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
		fmt.Println(paint(cDim, "  "+ct("cli.dashboard.opening")))
		openBrowser(addr)
		fmt.Println(paint(cDim, "  "+ct("cli.dashboard.ctrlc_stop")))
		fmt.Println()

		// Block until Ctrl+C or a POST /api/v1/shutdown.
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		return srv.Shutdown(shutCtx)
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

var configFavCmd = &cobra.Command{
	Use:     "favorites",
	Aliases: []string{"fav"},
	Short:   "Manage favorite VPN profiles",
}

var configFavListCmd = &cobra.Command{
	Use:   "list",
	Short: "List favorite profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Favorites []string `json:"favorites"`
			}{Favorites: p.Favorites})
		}
		if len(p.Favorites) == 0 {
			fmt.Println(styleDim.Render(ct("cli.favorites.none")))
			return nil
		}
		for _, name := range p.Favorites {
			fmt.Println("  " + styleGold.Render(sym("★", "*")) + "  " + styleBright.Render(name))
		}
		return nil
	},
}

var configFavAddCmd = &cobra.Command{
	Use:   "add <profile>",
	Short: "Add favorite profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := addFavorite(args[0]); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.favorites.added", args[0])))
		return nil
	},
}

var configFavRemoveCmd = &cobra.Command{
	Use:   "remove <profile>",
	Short: "Remove favorite profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := removeFavorite(args[0]); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.favorites.removed", args[0])))
		return nil
	},
}

var configDefaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Manage default CLI behavior",
}

var configDefaultsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				DefaultGroup string `json:"default_group"`
			}{DefaultGroup: p.DefaultGroup})
		}
		if p.DefaultGroup == "" {
			fmt.Println(styleDim.Render(ct("cli.defaults.none")))
			return nil
		}
		fmt.Println(tuiInfo(cf("cli.defaults.group", p.DefaultGroup)))
		return nil
	},
}

var configDefaultsSetGroupCmd = &cobra.Command{
	Use:   "set-group <group>",
	Short: "Set default group for 'kongtrol up'",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DefaultGroup = args[0]
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.defaults.group_set", args[0])))
		return nil
	},
}

var configLangCmd = &cobra.Command{
	Use:   "lang <es|en>",
	Short: "Set the CLI display language",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lang := strings.ToLower(strings.TrimSpace(args[0]))
		if lang != "es" && lang != "en" {
			return fmt.Errorf("%s", cf("cli.lang.invalid", args[0]))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.Language = lang
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.lang.set", lang)))
		return nil
	},
}

var configDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Manage the dashboard's local bind/port override",
}

var configDashboardShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the dashboard bind/port override",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Port int    `json:"port,omitempty"`
				Bind string `json:"bind,omitempty"`
			}{Port: p.DashboardPort, Bind: p.DashboardBind})
		}
		if p.DashboardPort == 0 && p.DashboardBind == "" {
			fmt.Println(styleDim.Render(ct("cli.config.dashboard.no_override")))
			return nil
		}
		if p.DashboardPort != 0 {
			fmt.Println(tuiInfo(cf("cli.config.dashboard.port", p.DashboardPort)))
		}
		if p.DashboardBind != "" {
			fmt.Println(tuiInfo(cf("cli.config.dashboard.bind", p.DashboardBind)))
		}
		return nil
	},
}

var configDashboardSetPortCmd = &cobra.Command{
	Use:   "set-port <port>",
	Short: "Override the dashboard's local port (applies on next restart)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil || port <= 0 || port > 65535 {
			return fmt.Errorf("%s", cf("cli.config.dashboard.invalid_port", args[0]))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardPort = port
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.config.dashboard.port_set", port)))
		return nil
	},
}

var configDashboardSetBindCmd = &cobra.Command{
	Use:   "set-bind <address>",
	Short: "Override the dashboard's bind address (applies on next restart)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bind := strings.TrimSpace(args[0])
		if err := config.ValidateDashboardBind(bind); err != nil {
			return fmt.Errorf("%s", cf("cli.config.dashboard.invalid_bind", bind))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardBind = bind
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.config.dashboard.bind_set", bind)))
		return nil
	},
}

var configDashboardResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Clear the dashboard bind/port override",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardPort = 0
		p.DashboardBind = ""
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(ct("cli.config.dashboard.reset")))
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate kongtrol.yaml without connecting",
	RunE: func(cmd *cobra.Command, args []string) error {
		var spin *spinner
		if !outputJSON {
			spin = newSpinner(ct("cli.config.validate.spinner"))
			spin.Start()
		}

		validatedCfg, err := config.Load(cfgPath)
		if spin != nil {
			spin.Stop()
		}
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Valid    bool `json:"valid"`
				Profiles int  `json:"profiles"`
				Policies int  `json:"policies"`
				Groups   int  `json:"groups"`
			}{
				Valid:    true,
				Profiles: len(validatedCfg.VPNs),
				Policies: len(validatedCfg.Policies),
				Groups:   len(validatedCfg.Groups),
			})
		}
		fmt.Println(tuiOK(styleBright.Render(ct("cli.config.validate.valid"))))
		fmt.Println("  " + styleDim.Render(fmt.Sprintf(
			ct("cli.config.validate.summary"),
			len(validatedCfg.VPNs), len(validatedCfg.Policies), len(validatedCfg.Groups),
		)))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
	configFavCmd.AddCommand(configFavListCmd, configFavAddCmd, configFavRemoveCmd)
	configDefaultsCmd.AddCommand(configDefaultsShowCmd, configDefaultsSetGroupCmd)
	configDashboardCmd.AddCommand(configDashboardShowCmd, configDashboardSetPortCmd, configDashboardSetBindCmd, configDashboardResetCmd)
	configCmd.AddCommand(configFavCmd, configDefaultsCmd, configLangCmd, configDashboardCmd)
}

// ── export ────────────────────────────────────────────────────────────────────

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print a sanitized config template (no secrets) for sharing with teammates",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
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
		// Piped / quiet output stays a bare, parseable version string.
		if outputQuiet || outputPlain || !isTerminal() {
			fmt.Println(version)
			return
		}
		PrintHeader(version)
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveProfiles returns the list of profile names from explicit args or a group.
func resolveProfiles(args []string, group string) ([]string, error) {
	if group != "" {
		if cfg == nil {
			return nil, fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
		}
		g, ok := cfg.Groups[group]
		if !ok {
			return nil, fmt.Errorf("%s", cf("cli.error.unknown_group", group))
		}
		return g.Profiles, nil
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", ct("cli.error.specify_profile_or_group"))
	}
	return args, nil
}

func resolveUpProfiles(args []string, group string, useFavorites bool) ([]string, error) {
	if len(args) > 0 || group != "" {
		return resolveProfiles(args, group)
	}
	prefs, err := loadPreferences()
	if err == nil {
		if useFavorites && len(prefs.Favorites) > 0 {
			return prefs.Favorites, nil
		}
		if prefs.DefaultGroup != "" {
			return resolveProfiles(nil, prefs.DefaultGroup)
		}
	}
	if useFavorites {
		return nil, fmt.Errorf("%s", ct("cli.error.no_favorites"))
	}
	return nil, fmt.Errorf("%s", ct("cli.error.specify_profile_or_group"))
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
//
// It prefers asking the daemon to shut down gracefully through its own API
// (POST /api/v1/shutdown) — that triggers the exact same cancellation path
// as Ctrl+C/SIGTERM, so kill-switch teardown, DNS restore, and history
// flush all run before the process exits. A bare kill skips every one of
// those, on every OS: Windows has no real cross-process SIGTERM equivalent
// (os.Process.Signal there only reliably implements Kill), so this was
// previously silent-but-unclean everywhere, not just on Windows.
// Falls back to a hard kill only if the API is unreachable (dashboard
// disabled) or the daemon doesn't exit within the grace period (wedged).
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
		fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.find", pid, err))
		return
	}

	base := daemonAPIBase()
	if probeDaemonAPI(base) {
		if err := daemonShutdown(base); err == nil {
			if waitForPIDFileGone(path, 10*time.Second) {
				fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.stopped", pid))
				return
			}
			fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.graceful_timeout", pid))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.graceful_failed", pid, err))
		}
	}

	// Fall back to a hard kill (API unreachable or the daemon never
	// finished shutting down within the grace period).
	if err := proc.Kill(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.stop", pid, err))
		return
	}
	_ = os.Remove(path)
	fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.stopped", pid))
}

// waitForPIDFileGone polls for the PID file to disappear, which the daemon
// does itself (removePIDFile) only after all of its deferred cleanup has
// already run — so its removal is a reliable, OS-agnostic signal that a
// graceful shutdown actually completed, without needing a platform-specific
// "is this PID still alive" check.
func waitForPIDFileGone(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return false
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tuiErr(err.Error()))
		os.Exit(1)
	}
}
