package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func (m upModel) filterLabel() string {
	switch m.filter {
	case upFilterConnected:
		return ct("cli.up.tui.filter.connected")
	case upFilterConnecting:
		return ct("cli.up.tui.filter.connecting")
	case upFilterError:
		return ct("cli.up.tui.filter.error")
	case upFilterDisconnected:
		return ct("cli.up.tui.filter.disconnected")
	default:
		return ct("cli.up.tui.filter.all")
	}
}

// passesFilter reports whether a tunnel in the given status should be shown
// under the current row filter.
func (m upModel) passesFilter(status vpn.Status) bool {
	switch m.filter {
	case upFilterConnected:
		return status == vpn.StatusConnected
	case upFilterConnecting:
		return status == vpn.StatusConnecting
	case upFilterError:
		return status == vpn.StatusError
	case upFilterDisconnected:
		return status != vpn.StatusConnected && status != vpn.StatusConnecting
	default:
		return true
	}
}

// filteredRows builds the visible tunnel table. When a Collector is
// available (daemon mode) it reads from its cached Snapshot() — populated
// on the collector's own poll goroutine — instead of calling
// adapter.Status()/TunnelInfo() directly, so a blocking adapter driver
// call can no longer stall this render path. status --watch (no local
// collector) falls back to querying its own adapters directly.
func (m upModel) filteredRows() []tunnelRow {
	var snapshot map[string]monitor.TunnelMetrics
	switch {
	case m.col != nil:
		snapshot = m.col.Snapshot()
	case m.remoteConnected:
		snapshot = m.remoteSnapshot
	}

	rows := make([]tunnelRow, 0, len(m.adapters))
	for _, name := range sortedAdapterNames(m.adapters) {
		status := vpn.StatusDisconnected
		ip, uptime := "—", "—"
		var attach *vpn.AttachState

		if snapshot != nil {
			if tm, ok := snapshot[name]; ok {
				status = tm.Status.Normalize()
				attach = tm.Attach
				if tm.AssignedIP != "" {
					ip = tm.AssignedIP
				}
				if !tm.ConnectedAt.IsZero() {
					uptime = formatUptime(m.now, tm.ConnectedAt)
				}
			}
		} else if adapter, ok := m.adapters[name]; ok {
			status = adapter.Status().Normalize()
			if r, ok := adapter.(vpn.AttachReporter); ok {
				attach = r.AttachState()
			}
			if info, err := adapter.TunnelInfo(); err == nil && info != nil {
				if info.AssignedIP != nil {
					ip = info.AssignedIP.String()
				}
				if !info.ConnectedAt.IsZero() {
					uptime = formatUptime(m.now, info.ConnectedAt)
				}
			}
		}

		if !m.passesFilter(status) {
			continue
		}
		rows = append(rows, tunnelRow{Name: name, Status: status, IP: ip, Uptime: uptime, Attach: attach})
	}
	return rows
}

func (m upModel) selectedProfile() (string, bool) {
	rows := m.filteredRows()
	if len(rows) == 0 || m.selected >= len(rows) {
		return "", false
	}
	return rows[m.selected].Name, true
}

// ── styles local to the up TUI ────────────────────────────────────────────────

var (
	upStyleFooter = lipgloss.NewStyle().Foreground(colDim)
	upStyleRule   = lipgloss.NewStyle().Foreground(colRule)
	upStyleDash   = lipgloss.NewStyle().Foreground(colSteel).Bold(true)
)

// renderTmuxBar draws the tmux-style status bar: session name on the left,
// the five status filters as numbered windows (active one highlighted), and
// live counters + clock on the right.
func (m upModel) renderTmuxBar(w int, rows []tunnelRow) string {
	bg := lipgloss.NewStyle().Background(colBarBg)
	seg := func(fg lipgloss.Color, bold bool, s string) string {
		st := lipgloss.NewStyle().Foreground(fg).Background(colBarBg)
		if bold {
			st = st.Bold(true)
		}
		return st.Render(s)
	}

	left := seg(colSignal, true, " [kongtrol] ")

	windowNames := []string{
		ct("cli.up.tui.filter.all"),
		ct("cli.up.tui.filter.connected"),
		ct("cli.up.tui.filter.connecting"),
		ct("cli.up.tui.filter.error"),
		ct("cli.up.tui.filter.disconnected"),
	}
	var windows strings.Builder
	for i, n := range windowNames {
		label := fmt.Sprintf(" %d:%s ", i, n)
		if upFilter(i) == m.filter {
			windows.WriteString(lipgloss.NewStyle().
				Foreground(colBarBg).Background(colSignal).Bold(true).
				Render(label))
		} else {
			windows.WriteString(seg(colDim, false, label))
		}
	}

	connected, _, _, _ := statusCounts(rows)
	_, ksActive, dnsOn, dnsActive := m.securityState()
	mark := func(on bool) (lipgloss.Color, string) {
		if on {
			return colSignal, sym("●", "*")
		}
		return colDim, sym("○", "o")
	}
	ksCol, ksMark := mark(ksActive)
	dnsCol, dnsMark := mark(dnsOn && dnsActive)
	right := seg(ksCol, false, "KS "+ksMark) +
		seg(colDim, false, " │ ") +
		seg(dnsCol, false, "DNS "+dnsMark) +
		seg(colDim, false, " │ ") +
		seg(colSignal, true, fmt.Sprintf("%d↑", connected)) +
		seg(colDim, false, " │ ") +
		seg(colMuted, false, m.now.Format("15:04:05")) +
		seg(colMuted, false, " ")

	used := lipgloss.Width(left) + lipgloss.Width(windows.String()) + lipgloss.Width(right)
	pad := max(1, w-used)
	return left + windows.String() + bg.Render(strings.Repeat(" ", pad)) + right
}

const upLogsMinHeight = 5
const upLogsHeaderHeight = 3 // blank separator + title line + rule

// renderPublicIPLine shows the current internet-facing IP as a reference
// point: when no tunnel is connected it's the "default" (no-VPN) address;
// once a tunnel is up it's labeled as the current (possibly VPN) egress IP
// so it isn't mistaken for the baseline.
//
// The view never looks the IP up itself: it shows the last opt-in lookup
// (made on tunnel connect/disconnect or `kongtrol check`), or a hint to
// turn the lookup on.
func (m *upModel) renderPublicIPLine(rowsData []tunnelRow) string {
	enabled, ip := m.publicIPState()
	if !enabled {
		return styleDim.Render(ct("cli.up.tui.public_ip.off"))
	}
	if ip == "" {
		return styleDim.Render(ct("cli.up.tui.public_ip.checking"))
	}
	connected, _, _, _ := statusCounts(rowsData)
	key := "cli.up.tui.public_ip.default"
	if connected > 0 {
		key = "cli.up.tui.public_ip.current"
	}
	return styleInfo.Render(cf(key, ip))
}

// publicIPState reads the daemon's state in a remote view, else this
// process's checker. No network call.
func (m *upModel) publicIPState() (enabled bool, ip string) {
	if m.remoteSec != nil {
		s := m.remoteSec
		if s.LeakCheck != nil {
			ip = s.LeakCheck.PublicIP
		}
		return s.PublicIPCheckEnabled || s.LeakCheckAvailable, ip
	}
	if m.leak == nil || !m.leak.Enabled() {
		return false, ""
	}
	if lr := m.leak.LastResult(); lr != nil {
		ip = lr.PublicIP
	}
	return true, ip
}

// renderStaticTop renders everything above the log panel: header, tunnel
// summary, security/watchdog lines, and the tunnel table. It never scrolls —
// it's always fully visible so the connection summary can't get pushed off
// screen by log output.
func (m *upModel) renderStaticTop(w int, rowsData []tunnelRow) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + kongTitle() + "  " + styleDim.Render(version) + " " + styleInfo.Render("[beta]") +
		"  " + styleDim.Render(sym("·", "-")+"  "+ct("cli.header.subtitle")) + "\n")
	b.WriteString("  " + upStyleRule.Render(strings.Repeat("─", max(0, w-2))) + "\n")
	if sessionGreetingLine != "" {
		b.WriteString("  " + styleDim.Render(sessionGreetingLine) + "\n")
	}
	if sessionLastUseLine != "" {
		b.WriteString("  " + styleDim.Render(sessionLastUseLine) + "\n")
	}
	b.WriteString("\n")

	b.WriteString(renderStatusSummary(rowsData) + "\n\n")

	ksOn, ksActive, dnsOn, dnsActive := m.securityState()
	b.WriteString("  " + renderSecurityLine(ksOn, ksActive, dnsOn, dnsActive, m.killSwitchDetail()))
	b.WriteString("\n")
	if line := policyNoticeLine(m.policyNotice, time.Now()); line != "" {
		b.WriteString("  " + tuiWarn(line) + "\n")
	}
	if line := m.renderPublicIPLine(rowsData); line != "" {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("  " + renderWatchdogLine())
	if m.dashURL != "" {
		b.WriteString("    " + styleInfo.Render(">  "+ct("cli.status.dashboard")+" → ") + upStyleDash.Render(m.dashURL))
	}
	b.WriteString("\n")
	if !m.daemonMode && !m.remoteConnected {
		b.WriteString("  " + styleWarn.Render(sym("⚠", "!")+"  "+ct("cli.up.tui.remote.no_daemon")) + "\n")
	}
	b.WriteString("\n")

	layout := computeTunnelTableLayout(rowsData, w)
	rule := upStyleRule.Render(strings.Repeat("─", layout.RuleW))
	b.WriteString(renderTunnelHeader(layout) + "\n")
	b.WriteString("  " + rule + "\n")
	for i, r := range rowsData {
		line := renderTunnelRow(layout, r)
		if m.focus == focusTable && i == m.selected {
			line = " " + stylePrompt.Render(">") + line[2:]
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("  " + rule + "\n")
	for _, line := range renderAttachRowLines(rowsData) {
		b.WriteString(line + "\n")
	}

	if m.notice != "" {
		b.WriteString("  " + styleInfo.Render(">  "+m.notice) + "\n")
	}

	return b.String()
}

// renderFooter renders the key-hint lines + tmux-style status bar pinned to
// the bottom of the screen.
func (m *upModel) renderFooter(w int, rowsData []tunnelRow) string {
	quitLine := ct("cli.up.tui.footer.quit_view")
	keysLine := ct("cli.up.tui.footer.keys_readonly")
	if m.daemonMode || m.apiBase != "" {
		keysLine = ct("cli.up.tui.footer.keys")
	}
	if m.daemonMode {
		quitLine = ct("cli.up.tui.footer.prefix") + styleBright.Render("kongtrol down") + ct("cli.up.tui.footer.suffix")
	}
	return "  " + upStyleFooter.Render(quitLine) + "\n" +
		"  " + upStyleFooter.Render(keysLine) + "\n" +
		m.renderTmuxBar(w, rowsData)
}

// renderLogsPanel renders the "Logs" section title (with a live/paused
// indicator) plus the scrollable viewport body.
func (m *upModel) renderLogsPanel(w int) string {
	focusMark := ""
	if m.focus == focusLogs {
		focusMark = "  " + stylePrompt.Render(">") + " " + upStyleFooter.Render(ct("cli.up.tui.logs.hint"))
	}
	state := styleOK.Render(sym("●", "*") + " " + ct("cli.up.tui.logs.live"))
	if !m.autoFollow {
		state = styleWarn.Render("||" + " " + ct("cli.up.tui.logs.paused"))
	}
	title := "  " + stylePrompt.Render(ct("cli.up.tui.logs.title")) + "  " + state + focusMark
	rule := upStyleRule.Render(strings.Repeat("─", max(0, w-2)))
	body := "  " + strings.ReplaceAll(m.logsVP.View(), "\n", "\n  ")
	return "\n" + title + "\n  " + rule + "\n" + body
}

func (m upModel) View() string {
	w := m.width
	if w < 60 {
		w = 80
	}

	// topCache/footerCache were rendered once already this Update cycle by
	// syncLogsViewport — reuse them instead of rendering the same content
	// again just to display it.
	logsPanel := m.renderLogsPanel(w)

	content := m.topCache + logsPanel
	if m.height > 0 {
		gap := m.height - lipgloss.Height(content) - lipgloss.Height(m.footerCache)
		if gap > 0 {
			content += strings.Repeat("\n", gap)
		}
	} else {
		content += "\n"
	}
	return content + m.footerCache
}

// securityState returns kill switch / DNS guard (enabled, active) for the
// security line. The daemon owns both features, so its reported state wins
// whenever it answers; the local objects are only a fallback when no daemon
// is reachable (this viewer process never engages them itself).
func (m *upModel) securityState() (ksOn, ksActive, dnsOn, dnsActive bool) {
	if m.remoteSec != nil {
		s := m.remoteSec
		return s.KillSwitchEnabled, s.KillSwitch, s.DNSGuardEnabled, s.DNSGuard
	}
	return m.ksOn, m.ks != nil && m.ks.IsEnabled(),
		m.dnsOn, m.dnsMgr != nil && m.dnsMgr.IsActive()
}

// killSwitchDetail is the daemon's split-tunnel kill switch detail, or nil.
func (m *upModel) killSwitchDetail() *security.KillSwitchStatus {
	if m.remoteSec != nil {
		return m.remoteSec.KillSwitchStatus
	}
	return nil
}
