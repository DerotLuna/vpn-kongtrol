package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

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
