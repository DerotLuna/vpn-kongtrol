package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── status ───────────────────────────────────────────────────────────────────

var statusWatch bool
var statusDashboard bool

var statusCmd = &cobra.Command{
	Use: "status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusDashboard {
			warnDeprecated("--dashboard", "kongtrol dashboard")
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
	// Deprecated: kongtrol dashboard opens the shared daemon's dashboard.
	statusCmd.Flags().BoolVar(&statusDashboard, "dashboard", false, "")
	_ = statusCmd.Flags().MarkHidden("dashboard")
	statusCmd.Flags().BoolVar(&alertBell, "alert-bell", false, ct("cli.flag.alert_bell"))
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
	// Opt-in public IP lookup (off by default) and the service it uses.
	PublicIPCheckEnabled bool   `json:"public_ip_check_enabled"`
	PublicIPCheckHost    string `json:"public_ip_check_host,omitempty"`
	// KillSwitchStatus: split-tunnel kill switch detail (protected /
	// blocking tunnels), rendered with killSwitchLabel.
	KillSwitchStatus *security.KillSwitchStatus `json:"kill_switch_status,omitempty"`
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
	LastError   string    `json:"last_error,omitempty"`
	LastErrorAt time.Time `json:"last_error_at,omitzero"`
	// LastErrorKey: i18n key of the guidance line for LastError, if any.
	LastErrorKey string `json:"last_error_key,omitempty"`
	// Attach is a provider adapter's attach phase (how it came up / what
	// the user must do in the app); render with its MessageKey.
	Attach *vpn.AttachState `json:"attach,omitempty"`
	// Health: does the server answer (WireGuard handshake, ADR-0030).
	Health *vpn.TunnelHealth `json:"health,omitempty"`
}

type statusSecurityJSON struct {
	KillSwitchEnabled       bool          `json:"kill_switch_enabled"`
	KillSwitchActive        bool          `json:"kill_switch_active"`
	DNSGuardEnabled         bool          `json:"dns_guard_enabled"`
	DNSGuardActive          bool          `json:"dns_guard_active"`
	LeakDetectionConfigured bool          `json:"leak_detection_configured"`
	LeakCheckAvailable      bool          `json:"leak_check_available"`
	LeakCheck               *apiLeakCheck `json:"leak_check,omitempty"`
	PublicIPCheckEnabled    bool          `json:"public_ip_check_enabled"`
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
		Tunnels:     make([]statusTunnelJSON, 0, adapters.Len()),
		Security: statusSecurityJSON{
			KillSwitchEnabled:       cfg.Security.KillSwitch.Enabled,
			DNSGuardEnabled:         cfg.Security.DNSGuard.Enabled,
			LeakDetectionConfigured: cfg.PublicIPCheckEnabled(),
			LeakCheckAvailable:      leak.Enabled(),
			PublicIPCheckEnabled:    leak.Enabled(),
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
		report.Security.PublicIPCheckEnabled = sec.PublicIPCheckEnabled || sec.LeakCheckAvailable
		for _, t := range tunnels {
			item := statusTunnelJSON{
				Name:         t.Name,
				Status:       string(t.Status.Normalize()),
				AssignedIP:   t.AssignedIP,
				LastError:    t.LastError,
				LastErrorAt:  t.LastErrorAt,
				LastErrorKey: t.LastErrorKey,
				Attach:       t.Attach,
				Health:       t.Health,
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
	for _, name := range adapters.Names() {
		adapter, _ := adapters.Get(name)
		status := adapter.Status().Normalize()
		item := statusTunnelJSON{
			Name:   name,
			Status: string(status),
		}
		if r, ok := adapter.(vpn.AttachReporter); ok {
			item.Attach = r.AttachState()
		}
		if r, ok := adapter.(vpn.HealthReporter); ok {
			item.Health = r.TunnelHealth()
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
	client := daemonctl.HTTPClient(1200 * time.Millisecond)
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
	return fetchDaemonSnapshotAt(dashboardURL())
}

// fetchDaemonSnapshotAt reads tunnels + security state from the daemon at base.
func fetchDaemonSnapshotAt(base string) ([]monitor.TunnelMetrics, *apiSecurityStatus, error) {
	client := daemonctl.HTTPClient(1200 * time.Millisecond)

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

	sec, err := fetchDaemonSecurity(client, base)
	if err != nil {
		return nil, nil, err
	}
	return tunnels, sec, nil
}

// fetchDaemonSecurity reads the daemon's kill switch / DNS guard state —
// the single source both `status` and the live view use.
func fetchDaemonSecurity(client *http.Client, base string) (*apiSecurityStatus, error) {
	sReq, err := daemonRequest(http.MethodGet, base+"/api/v1/security/status", nil)
	if err != nil {
		return nil, err
	}
	sResp, err := client.Do(sReq)
	if err != nil {
		return nil, err
	}
	defer sResp.Body.Close()
	if sResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("security status unavailable")
	}
	sec := &apiSecurityStatus{}
	if err := json.NewDecoder(sResp.Body).Decode(sec); err != nil {
		return nil, err
	}
	return sec, nil
}

func printStatus() {
	now := time.Now()
	width := terminalWidth()

	tunnels, sec, daemonErr := fetchDaemonSnapshot()
	if daemonErr == nil {
		history, _ := fetchDaemonHistory()
		seen := make(map[string]bool, len(tunnels))
		rows := make([]tunnelRow, 0, len(tunnels)+adapters.Len())
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
		for _, name := range adapters.Names() {
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
		fmt.Println("  " + renderSecurityLine(sec.KillSwitchEnabled, sec.KillSwitch, sec.DNSGuardEnabled, sec.DNSGuard, sec.KillSwitchStatus))
		fmt.Println("  " + renderLeakCheckLine(sec.LeakDetectionConfigured, sec.LeakCheckAvailable, sec.LeakCheck))
		fmt.Println("  " + renderWatchdogLine())
		fmt.Println()
		fmt.Println(renderTunnelHeader(layout))
		fmt.Println("  " + sep)
		for _, r := range rows {
			fmt.Println(renderTunnelRow(layout, r))
		}
		fmt.Println("  " + sep)
		for _, line := range renderAttachLines(tunnels) {
			fmt.Println(line)
		}
		for _, line := range renderStatusRoutesLines(dashboardURL()) {
			fmt.Println(line)
		}
		for _, line := range renderLastErrorLines(tunnels, now) {
			fmt.Println(line)
		}
		if hint := nothingConnectedHint(rows); hint != "" {
			fmt.Println(hint)
		}
		return
	}

	rows := make([]tunnelRow, 0, adapters.Len())
	history := map[string]monitor.ProfileHistory{}
	if col != nil {
		history = col.HistorySnapshot()
	}
	for _, name := range adapters.Names() {
		adapter, _ := adapters.Get(name)
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
	fmt.Println("  " + renderLeakCheckLine(cfg.PublicIPCheckEnabled(), leak.Enabled(), leakCheck))
	fmt.Println("  " + renderWatchdogLine())
	fmt.Println()
	fmt.Println(renderTunnelHeader(layout))
	fmt.Println("  " + sep)
	for _, r := range rows {
		fmt.Println(renderTunnelRow(layout, r))
	}
	fmt.Println("  " + sep)
	local := make([]monitor.TunnelMetrics, 0, adapters.Len())
	for _, name := range adapters.Names() {
		if m, ok := localTunnelState(name); ok {
			local = append(local, m)
		}
	}
	for _, line := range renderAttachLines(local) {
		fmt.Println(line)
	}
	if hint := nothingConnectedHint(rows); hint != "" {
		fmt.Println(hint)
	}
	if line := updateLine(); line != "" {
		fmt.Println(tuiInfo(paint(cDim, line)))
	}
}

// renderAttachLines lists, under the status table, each tunnel's health
// line ("Sin respuesta del servidor · …", ADR-0030) and each provider
// tunnel's attach message ("NordVPN ya conectado · kongtrol lo toma",
// "Esperando conexión en Surfshark…"), problems and waiting highlighted.
func renderAttachLines(tunnels []monitor.TunnelMetrics) []string {
	var lines []string
	for _, t := range tunnels {
		if line := healthLine(t.Health); line != "" {
			label := paintBold(cBright, t.Name) + "  " + line
			if t.Health.Problem() {
				lines = append(lines, "  "+tuiWarn(label))
			} else {
				lines = append(lines, "  "+tuiInfo(paint(cDim, label)))
			}
		}
		line := attachLine(t.Attach)
		if line == "" {
			continue
		}
		label := paintBold(cBright, t.Name) + "  " + line
		if t.Attach.Waiting() {
			lines = append(lines, "  "+tuiWarn(label))
		} else {
			lines = append(lines, "  "+tuiInfo(paint(cDim, label)))
		}
	}
	return lines
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

	// Deprecated --dashboard: only the shared daemon serves the dashboard
	// (kongtrol dashboard starts it); show its address when it answers.
	var dashURL string
	if base := daemonAPIBase(); statusDashboard && probeDaemonAPI(base) {
		dashURL = base
	}

	runUpTUI(ctx, cancel, adapters.Snapshot(), ks, dnsMgr, dashURL, false)
	return nil
}

// renderLastErrorLines lists, under the status table, why each tunnel that
// is not connected failed its last connect, so a failed async connect
// (dashboard, tray, `up`) is visible without opening the daemon log.
func renderLastErrorLines(tunnels []monitor.TunnelMetrics, now time.Time) []string {
	var lines []string
	for _, t := range tunnels {
		if t.LastError == "" || t.Status.Normalize() == vpn.StatusConnected {
			continue
		}
		ago := formatElapsedCompact(now.Sub(t.LastErrorAt))
		lines = append(lines, "  "+tuiErr(paintBold(cBright, t.Name)+"  "+cf("cli.status.last_error", ago, tunnelErrorText(t))))
	}
	return lines
}
