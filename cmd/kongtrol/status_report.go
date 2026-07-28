package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

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
