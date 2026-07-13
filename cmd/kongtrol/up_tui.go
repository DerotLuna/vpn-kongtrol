package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── Bubble Tea model for the live daemon view ─────────────────────────────────

type upTickMsg time.Time
type upQuitMsg struct{}

type upModel struct {
	adapters map[string]vpn.VPNAdapter
	ks       security.KillSwitch
	dnsMgr   *monitor.DNSManager
	dashURL  string
	cancel   context.CancelFunc
	now      time.Time
	width    int
	height   int
}

func newUpModel(
	adapters map[string]vpn.VPNAdapter,
	ks security.KillSwitch,
	dnsMgr *monitor.DNSManager,
	dashURL string,
	cancel context.CancelFunc,
) upModel {
	return upModel{
		adapters: adapters,
		ks:       ks,
		dnsMgr:   dnsMgr,
		dashURL:  dashURL,
		cancel:   cancel,
		now:      time.Now(),
		width:    80,
	}
}

func (m upModel) Init() tea.Cmd {
	return upTick()
}

func upTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return upTickMsg(t) })
}

func (m upModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case upTickMsg:
		m.now = time.Time(msg)
		return m, upTick()

	case upQuitMsg:
		// External signal received — exit cleanly.
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Cancel the parent context so all defers in upCmd fire.
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// ── styles local to the up TUI ────────────────────────────────────────────────

var (
	upStyleBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("136")).
			Padding(0, 1)

	upStyleFooter = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	upStyleRule   = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
	upStyleKSOn   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	upStyleKSOff  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	upStyleDNSOn  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	upStyleDNSOff = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	upStyleDash   = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
)

func (m upModel) View() string {
	w := m.width
	if w < 60 {
		w = 80
	}

	var b strings.Builder

	// ── header ────────────────────────────────────────────────────────────────
	b.WriteString("\n")
	b.WriteString("  " + kongTitle() + "  " + styleDim.Render(version) + "\n")
	b.WriteString("\n")

	// ── tunnel table ──────────────────────────────────────────────────────────
	const (
		nameW   = 20
		statusW = 14
		ipW     = 18
	)
	ruleW := nameW + statusW + ipW + 14
	if ruleW > w-4 {
		ruleW = w - 4
	}
	rule := upStyleRule.Render(strings.Repeat("─", ruleW))

	rows := []string{
		"  " + fmt.Sprintf("%s %s %s %s",
			styleStatusHdr.Render(pad("TUNNEL", nameW)),
			styleStatusHdr.Render(pad("STATUS", statusW)),
			styleStatusHdr.Render(pad("IP", ipW)),
			styleStatusHdr.Render("UPTIME")),
		"  " + rule,
	}

	for name, adapter := range m.adapters {
		status := adapter.Status()
		ip := styleDim.Render("—")
		uptime := styleDim.Render("—")

		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			if info.AssignedIP != nil {
				ip = styleStatusIP.Render(info.AssignedIP.String())
			}
			if !info.ConnectedAt.IsZero() {
				d := m.now.Sub(info.ConnectedAt)
				if d < 0 {
					d = 0
				}
				d = d.Round(time.Second)
				h := int(d.Hours())
				min := int(d.Minutes()) % 60
				sec := int(d.Seconds()) % 60
				uptime = styleStatusTime.Render(fmt.Sprintf("%dh %02dm %02ds", h, min, sec))
			}
		}

		rows = append(rows, fmt.Sprintf("  %s %s %s %s %s",
			statusDot(status),
			styleStatusName.Render(pad(name, nameW)),
			pad(statusLabel(status), statusW+30),
			pad(ip, ipW+20),
			uptime))
	}
	rows = append(rows, "  "+rule)

	for _, row := range rows {
		b.WriteString(row + "\n")
	}
	b.WriteString("\n")

	// ── status bar (kill switch + dashboard) ─────────────────────────────────
	var statusParts []string
	if m.ks != nil {
		if m.ks.IsEnabled() {
			statusParts = append(statusParts,
				upStyleKSOn.Render("⬡  Kill switch  ACTIVE"))
		} else {
			statusParts = append(statusParts,
				upStyleKSOff.Render("⬡  Kill switch  off"))
		}
	}
	if m.dnsMgr != nil {
		if m.dnsMgr.IsActive() {
			statusParts = append(statusParts,
				upStyleDNSOn.Render("⬡  DNS Guard  ACTIVE"))
		} else {
			statusParts = append(statusParts,
				upStyleDNSOff.Render("⬡  DNS Guard  off"))
		}
	}
	if m.dashURL != "" {
		statusParts = append(statusParts,
			styleInfo.Render("▸  Dashboard → ")+upStyleDash.Render(m.dashURL))
	}
	if len(statusParts) > 0 {
		b.WriteString("  " + strings.Join(statusParts, "    ") + "\n")
	}
	b.WriteString("\n")

	// ── footer ────────────────────────────────────────────────────────────────
	b.WriteString("  " + upStyleRule.Render(strings.Repeat("─", ruleW)) + "\n")
	b.WriteString("  " + upStyleFooter.Render(
		"q / Ctrl+C  →  stop daemon  ·  VPN stays connected  ·  run "+
			styleBright.Render("kongtrol down")+" to disconnect") + "\n")

	return b.String()
}

// runUpTUI starts the Bubble Tea daemon view. It blocks until the user quits
// (q / Ctrl+C) or ctx is cancelled by an external signal (SIGTERM / kill).
// When the user quits interactively, cancel() is called to trigger upCmd's defers.
func runUpTUI(ctx context.Context, cancel context.CancelFunc, adapters map[string]vpn.VPNAdapter, ks security.KillSwitch, dnsMgr *monitor.DNSManager, dashURL string) {
	if !isTerminal {
		// Non-interactive: just block until cancelled.
		<-ctx.Done()
		return
	}

	model := newUpModel(adapters, ks, dnsMgr, dashURL, cancel)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Forward external cancellation (SIGTERM / kill) into Bubble Tea.
	go func() {
		<-ctx.Done()
		p.Send(upQuitMsg{})
	}()

	if _, err := p.Run(); err != nil {
		// If Bubble Tea fails (e.g. unsupported terminal), fall back.
		<-ctx.Done()
	}
}
