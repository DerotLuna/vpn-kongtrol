package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── active-program registry ───────────────────────────────────────────────────
// While the Bubble Tea daemon view owns the terminal (alt screen), any
// out-of-band writes to stdout/stderr (alerts, watchdog events, leak
// notifications) would corrupt the display. emitAlert routes through here
// instead, feeding lines into the TUI's own scrollable log panel.

var (
	tuiProgramMu sync.Mutex
	tuiProgram   *tea.Program
)

func setActiveTUIProgram(p *tea.Program) {
	tuiProgramMu.Lock()
	tuiProgram = p
	tuiProgramMu.Unlock()
}

func clearActiveTUIProgram(p *tea.Program) {
	tuiProgramMu.Lock()
	if tuiProgram == p {
		tuiProgram = nil
	}
	tuiProgramMu.Unlock()
}

// sendTUILog delivers a rendered log line to the active TUI, if any. It
// returns false when no TUI owns the terminal, so the caller can fall back
// to printing directly.
func sendTUILog(line string) bool {
	tuiProgramMu.Lock()
	p := tuiProgram
	tuiProgramMu.Unlock()
	if p == nil {
		return false
	}
	p.Send(upLogMsg{line: line})
	return true
}

const upLogMaxLines = 500

// ── Bubble Tea model for the live daemon view ─────────────────────────────────

type upTickMsg time.Time
type upQuitMsg struct{}
type upActionDoneMsg struct {
	profile string
	action  string
	err     error
}
type upLogMsg struct{ line string }

type upFilter int

const (
	upFilterAll upFilter = iota
	upFilterConnected
	upFilterConnecting
	upFilterError
	upFilterDisconnected
)

type upFocus int

const (
	focusTable upFocus = iota
	focusLogs
)

type upModel struct {
	adapters map[string]vpn.VPNAdapter
	ks       security.KillSwitch
	ksOn     bool
	dnsMgr   *monitor.DNSManager
	dnsOn    bool
	dashURL  string
	cancel   context.CancelFunc
	now      time.Time
	width    int
	height   int
	selected int
	filter   upFilter
	busy     bool
	notice   string

	focus      upFocus
	logs       []string
	logsVP     viewport.Model
	logsVPInit bool
	autoFollow bool

	// daemonMode is true for `kongtrol up` (this process owns the daemon
	// lifecycle, so quitting stops it) and false for `kongtrol status --watch`
	// (a viewer — quitting only closes the view, it never owns a daemon
	// lifecycle here).
	daemonMode bool

	// apiBase is set (status --watch only) when a `kongtrol up` daemon's API
	// server was detected on the well-known local address. Connect/disconnect/
	// reconnect actions are proxied through it via HTTP instead of running
	// in-process, so the real daemon (with its watchdog and DNS/kill-switch
	// bookkeeping) stays the single owner of every tunnel. When empty, no
	// daemon was found and actions are disabled to avoid spawning a second,
	// uncoordinated adapter instance.
	apiBase string

	leak       *security.LeakTester
	publicIP   string
	publicIPAt time.Time
	ipChecking bool

	// col is the daemon's already-running metrics collector (nil in
	// status --watch, a separate viewer process with no collector of its
	// own). When set, filteredRows reads its cached Snapshot() instead of
	// calling adapter.Status()/TunnelInfo() directly on every render —
	// those adapter calls now happen only once per collector poll
	// interval, on the collector's own goroutine, decoupled from the
	// render loop.
	col *monitor.Collector
	// changes is notified by col whenever a collection pass observes a
	// real tunnel-state change, letting the TUI react to state changes
	// as they happen instead of only discovering them on the next 1s
	// clock tick.
	changes <-chan struct{}

	// remoteSnapshot/remoteConnected hold the live tunnel state streamed
	// from a real `kongtrol up` daemon's WebSocket feed, used only in
	// status --watch (daemonMode == false, col == nil — this process has
	// no collector of its own). Without this, filteredRows would have to
	// fall back to querying this process's own never-connected adapter
	// instances, silently showing "disconnected" even while a real daemon
	// elsewhere has active tunnels.
	remoteSnapshot  map[string]monitor.TunnelMetrics
	remoteConnected bool

	// topCache/footerCache hold the last-rendered static sections, computed
	// once per Update and reused by View (which bubbletea always calls right
	// after Update with the same model) — avoids paying for two full
	// lipgloss renders of the same content every tick/keystroke.
	topCache    string
	footerCache string
	// logsDirty marks that m.logs changed since the last viewport.SetContent
	// call, so it isn't rebuilt (Join over up to 500 lines) on every tick.
	logsDirty bool
}

func newUpModel(
	adapters map[string]vpn.VPNAdapter,
	ks security.KillSwitch,
	ksOn bool,
	dnsMgr *monitor.DNSManager,
	dnsOn bool,
	dashURL string,
	cancel context.CancelFunc,
	daemonMode bool,
	leak *security.LeakTester,
	apiBase string,
	col *monitor.Collector,
	changes <-chan struct{},
) upModel {
	m := upModel{
		adapters:   adapters,
		ks:         ks,
		ksOn:       ksOn,
		dnsMgr:     dnsMgr,
		dnsOn:      dnsOn,
		dashURL:    dashURL,
		cancel:     cancel,
		now:        time.Now(),
		width:      80,
		filter:     upFilterAll,
		autoFollow: true,
		leak:       leak,
		daemonMode: daemonMode,
		apiBase:    apiBase,
		col:        col,
		changes:    changes,
	}
	// Pre-populate the render caches so the very first View() (which
	// bubbletea can call before any Update) isn't blank.
	m.syncLogsViewport()
	return m
}

func (m upModel) Init() tea.Cmd {
	cmds := []tea.Cmd{upTick()}
	if m.leak != nil {
		cmds = append(cmds, checkPublicIPCmd(m.leak))
	}
	if c := waitForCollectorChange(m.changes); c != nil {
		cmds = append(cmds, c)
	}
	return tea.Batch(cmds...)
}

func upTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return upTickMsg(t) })
}

// upCollectorChangedMsg is delivered when the daemon's Collector observes a
// real tunnel-state change, so the TUI can react immediately instead of
// waiting for the next 1s clock tick to notice.
type upCollectorChangedMsg struct{}

// waitForCollectorChange returns a tea.Cmd that blocks on ch and reports a
// single change, re-armed by the Update case that handles it. Returns nil
// when ch is nil (status --watch, no collector running in this process).
func waitForCollectorChange(ch <-chan struct{}) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		<-ch
		return upCollectorChangedMsg{}
	}
}

// publicIPRefreshInterval controls how often the reference public IP line
// re-checks — it's a real network call, so this stays well below the 1s
// status tick.
const publicIPRefreshInterval = 30 * time.Second

type upPublicIPMsg struct {
	ip  string
	err error
}

// checkPublicIPCmd performs a single (blocking, in its own goroutine) public
// IP lookup via the same LeakTester used for leak detection.
func checkPublicIPCmd(leak *security.LeakTester) tea.Cmd {
	return func() tea.Msg {
		result := leak.CheckNow()
		if result.PublicIP == "" {
			return upPublicIPMsg{err: fmt.Errorf("%s", result.Reason)}
		}
		return upPublicIPMsg{ip: result.PublicIP}
	}
}

func publicIPTick() tea.Cmd {
	return tea.Tick(publicIPRefreshInterval, func(t time.Time) tea.Msg { return upPublicIPTickMsg(t) })
}

type upPublicIPTickMsg time.Time

// upRemoteStateMsg reports a change in status --watch's connection to a
// real `kongtrol up` daemon: found/streaming (connected, with a fresh
// snapshot and the apiBase to proxy actions through) or lost (connected
// == false, back to probing).
type upRemoteStateMsg struct {
	connected bool
	apiBase   string
	snapshot  map[string]monitor.TunnelMetrics
}

func (m upModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case upTickMsg:
		m.now = time.Time(msg)
		cmd = upTick()

	case upPublicIPTickMsg:
		if m.leak != nil && !m.ipChecking {
			m.ipChecking = true
			cmd = checkPublicIPCmd(m.leak)
		}

	case upPublicIPMsg:
		m.ipChecking = false
		if msg.err == nil {
			m.publicIP = msg.ip
			m.publicIPAt = m.now
		}
		cmd = publicIPTick()

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
		case "tab":
			if m.focus == focusTable {
				m.focus = focusLogs
			} else {
				m.focus = focusTable
			}
		case "up", "k", "down", "j", "pgup", "pgdown", "g", "G":
			if m.focus == focusLogs {
				m.logsVP, cmd = m.logsVP.Update(msg)
				m.autoFollow = m.logsVP.AtBottom()
			} else {
				switch msg.String() {
				case "up", "k":
					if m.selected > 0 {
						m.selected--
					}
				case "down", "j":
					rows := m.filteredRows()
					if len(rows) > 0 && m.selected < len(rows)-1 {
						m.selected++
					}
				}
			}
		case "f":
			if m.focus != focusLogs {
				m.filter = (m.filter + 1) % 5
				m.selected = 0
				m.notice = cf("cli.up.tui.filter_changed", m.filterLabel())
			}
		case "o":
			if m.focus != focusLogs && m.dashURL != "" {
				if err := openBrowser(m.dashURL); err != nil {
					m.notice = cf("cli.dashboard.open_failed", err)
				} else {
					m.notice = ct("cli.up.tui.opened_dashboard")
				}
			}
		case "c":
			if m.focus == focusLogs || m.busy {
				break
			}
			if !m.daemonMode && m.apiBase == "" {
				m.notice = ct("cli.up.tui.action_no_daemon")
				break
			}
			if name, ok := m.selectedProfile(); ok {
				m.busy = true
				m.notice = cf("cli.up.tui.action_running", ct("cli.up.tui.action.connect"), name)
				m.appendLog(tuiInfo(m.notice))
				cmd = runTUIProfileAction(name, "connect", m.apiBase)
			}
		case "d":
			if m.focus == focusLogs || m.busy {
				break
			}
			if !m.daemonMode && m.apiBase == "" {
				m.notice = ct("cli.up.tui.action_no_daemon")
				break
			}
			if name, ok := m.selectedProfile(); ok {
				m.busy = true
				m.notice = cf("cli.up.tui.action_running", ct("cli.up.tui.action.disconnect"), name)
				m.appendLog(tuiInfo(m.notice))
				cmd = runTUIProfileAction(name, "disconnect", m.apiBase)
			}
		case "r":
			if m.focus == focusLogs || m.busy {
				break
			}
			if !m.daemonMode && m.apiBase == "" {
				m.notice = ct("cli.up.tui.action_no_daemon")
				break
			}
			if name, ok := m.selectedProfile(); ok {
				m.busy = true
				m.notice = cf("cli.up.tui.action_running", ct("cli.up.tui.action.reconnect"), name)
				m.appendLog(tuiInfo(m.notice))
				cmd = runTUIProfileAction(name, "reconnect", m.apiBase)
			}
		default:
			if m.focus == focusLogs {
				m.logsVP, cmd = m.logsVP.Update(msg)
				m.autoFollow = m.logsVP.AtBottom()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case upActionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.notice = cf("cli.up.tui.action_failed", msg.action, msg.profile, msg.err)
			m.appendLog(tuiErr(m.notice))
		} else {
			m.notice = cf("cli.up.tui.action_done", msg.action, msg.profile)
			m.appendLog(tuiOK(m.notice))
		}

	case upLogMsg:
		m.appendLog(msg.line)

	case upCollectorChangedMsg:
		cmd = waitForCollectorChange(m.changes)

	case upRemoteStateMsg:
		m.remoteConnected = msg.connected
		if msg.connected {
			m.apiBase = msg.apiBase
			m.remoteSnapshot = msg.snapshot
		} else {
			m.apiBase = ""
		}
	}

	m.syncLogsViewport()
	m.clampSelected()
	return m, cmd
}

// appendLog records a rendered line in the log panel's history, capped at
// upLogMaxLines.
func (m *upModel) appendLog(line string) {
	m.logs = append(m.logs, line)
	if len(m.logs) > upLogMaxLines {
		m.logs = m.logs[len(m.logs)-upLogMaxLines:]
	}
	m.logsDirty = true
}

// clampSelected keeps the selected row valid after the filtered row set
// shrinks (a filter change, or a tunnel dropping out of the current filter).
func (m *upModel) clampSelected() {
	rows := m.filteredRows()
	if len(rows) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(rows) {
		m.selected = len(rows) - 1
	}
}

// syncLogsViewport keeps the log viewport's dimensions and content in step
// with the terminal size and the static (non-scrolling) content surrounding
// it, so the tunnel summary/table stay pinned while only the log panel
// scrolls. It also renders and caches the static top/footer sections once
// per Update, and only rebuilds the viewport's content when the log history
// actually changed — View() reuses both, instead of paying for two full
// lipgloss renders and a log-content rebuild on every 1s tick/keystroke.
func (m *upModel) syncLogsViewport() {
	w := m.width
	if w < 60 {
		w = 80
	}
	rowsData := m.filteredRows()
	m.topCache = m.renderStaticTop(w, rowsData)
	m.footerCache = m.renderFooter(w, rowsData)

	reserved := lipgloss.Height(m.topCache) + lipgloss.Height(m.footerCache) + upLogsHeaderHeight
	height := upLogsMinHeight
	if m.height > 0 {
		if h := m.height - reserved; h > height {
			height = h
		}
	}
	vpWidth := max(1, w-2)
	if !m.logsVPInit {
		m.logsVP = viewport.New(vpWidth, height)
		m.logsVP.SetContent(strings.Join(m.logs, "\n"))
		m.logsVPInit = true
		m.logsDirty = false
	} else {
		m.logsVP.Width = vpWidth
		m.logsVP.Height = height
		if m.logsDirty {
			m.logsVP.SetContent(strings.Join(m.logs, "\n"))
			m.logsDirty = false
		}
	}
	if m.autoFollow {
		m.logsVP.GotoBottom()
	}
}

// runTUIProfileAction performs a connect/disconnect/reconnect action. When
// apiBase is set (status --watch with a live daemon detected), it proxies
// the action through the daemon's REST API instead of driving the adapter
// in-process — this process's adapters map is a separate instance from the
// daemon's, so acting on it directly would spawn a second, uncoordinated
// tunnel rather than controlling the one the daemon already owns.
func runTUIProfileAction(name, action, apiBase string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		var err error
		if apiBase != "" {
			switch action {
			case "connect":
				err = daemonConnect(apiBase, name)
			case "disconnect":
				err = daemonDisconnect(apiBase, name)
			case "reconnect":
				err = daemonReconnect(apiBase, name)
			}
			return upActionDoneMsg{profile: name, action: action, err: err}
		}
		switch action {
		case "connect":
			err = connectProfile(ctx, name)
		case "disconnect":
			err = disconnectProfile(ctx, name)
		case "reconnect":
			if err = disconnectProfile(ctx, name); err == nil {
				err = connectProfile(ctx, name)
			}
		}
		return upActionDoneMsg{profile: name, action: action, err: err}
	}
}

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

		if snapshot != nil {
			if tm, ok := snapshot[name]; ok {
				status = tm.Status.Normalize()
				if tm.AssignedIP != "" {
					ip = tm.AssignedIP
				}
				if !tm.ConnectedAt.IsZero() {
					uptime = formatUptime(m.now, tm.ConnectedAt)
				}
			}
		} else if adapter, ok := m.adapters[name]; ok {
			status = adapter.Status().Normalize()
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
		rows = append(rows, tunnelRow{Name: name, Status: status, IP: ip, Uptime: uptime})
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
	ksActive := m.ks != nil && m.ks.IsEnabled()
	dnsActive := m.dnsMgr != nil && m.dnsMgr.IsActive()
	mark := func(on bool) (lipgloss.Color, string) {
		if on {
			return colSignal, sym("●", "*")
		}
		return colDim, sym("○", "o")
	}
	ksCol, ksMark := mark(m.ksOn && ksActive)
	dnsCol, dnsMark := mark(m.dnsOn && dnsActive)
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
func (m *upModel) renderPublicIPLine(rowsData []tunnelRow) string {
	if m.leak == nil {
		return ""
	}
	if m.publicIP == "" {
		return styleDim.Render(ct("cli.up.tui.public_ip.checking"))
	}
	connected, _, _, _ := statusCounts(rowsData)
	key := "cli.up.tui.public_ip.default"
	if connected > 0 {
		key = "cli.up.tui.public_ip.current"
	}
	return styleInfo.Render(cf(key, m.publicIP))
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

	ksActive := m.ks != nil && m.ks.IsEnabled()
	dnsActive := m.dnsMgr != nil && m.dnsMgr.IsActive()
	b.WriteString("  " + renderSecurityLine(m.ksOn, ksActive, m.dnsOn, dnsActive))
	b.WriteString("\n")
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

// runUpTUI starts the Bubble Tea daemon view. It blocks until the user quits
// (q / Ctrl+C) or ctx is cancelled by an external signal (SIGTERM / kill).
// When the user quits interactively, cancel() is called to trigger upCmd's defers.
// daemonMode distinguishes `up` (this process owns the daemon lifecycle) from
// `status --watch` (a read-only viewer with no daemon to tear down) — it only
// changes the footer copy, not the behavior.
func runUpTUI(ctx context.Context, cancel context.CancelFunc, adapters map[string]vpn.VPNAdapter, ks security.KillSwitch, dnsMgr *monitor.DNSManager, dashURL string, daemonMode bool) {
	if !isTerminal() {
		// Non-interactive: just block until cancelled.
		<-ctx.Done()
		return
	}

	// In daemon mode, `col` (the global metrics collector, already started
	// in the `up` command's setup) is this process's own tunnel state —
	// subscribe so the TUI reads its cache and wakes on real changes
	// instead of polling adapters itself. status --watch has no local
	// collector (it's a separate viewer process), so it keeps its own
	// adapter-based fallback in filteredRows.
	var modelCol *monitor.Collector
	var changes <-chan struct{}
	if daemonMode && col != nil {
		modelCol = col
		var unsubscribe func()
		changes, unsubscribe = col.Subscribe()
		defer unsubscribe()
	}

	model := newUpModel(adapters, ks, cfg.Security.KillSwitch.Enabled, dnsMgr, cfg.Security.DNSGuard.Enabled, dashURL, cancel, daemonMode, leak, "", modelCol, changes)
	// No mouse mode: on classic Windows consoles (conhost/PowerShell) mouse
	// tracking doesn't override "QuickEdit Mode" — a click inside the window
	// still freezes the whole process in a text-selection state. Keyboard
	// scrolling (↑/↓/pgup/pgdown/g/G) covers the log panel without inviting
	// mouse use over the window.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Register this program as the alerts/log sink so emitAlert et al. feed
	// the scrollable log panel instead of writing to stderr and corrupting
	// the alt-screen display.
	setActiveTUIProgram(p)
	defer clearActiveTUIProgram(p)

	// status --watch has no local collector — instead of a one-shot probe
	// at startup (which would go stale forever if the daemon starts later,
	// or silently keep showing this process's own disconnected adapters if
	// none was found yet), keep discovering/streaming the real daemon's
	// live state for the life of the session.
	if !daemonMode {
		go watchRemoteDaemon(ctx, p)
	}

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

// watchRemoteDaemon runs for the lifetime of a status --watch session. It
// repeatedly probes for a live `kongtrol up` daemon and, once found, streams
// its live tunnel snapshot over the same WebSocket feed the dashboard uses,
// retrying on disconnect — instead of a single startup probe that goes
// stale if the daemon appears later or drops and comes back.
func watchRemoteDaemon(ctx context.Context, p *tea.Program) {
	const retryDelay = 3 * time.Second
	base := daemonAPIBase()

	for ctx.Err() == nil {
		if !probeDaemonAPI(base) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			continue
		}
		streamRemoteTunnels(ctx, p, base)
		p.Send(upRemoteStateMsg{connected: false})
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

// streamRemoteTunnels dials the daemon's live metrics WebSocket and
// forwards every snapshot into the TUI until the connection drops or ctx
// is cancelled.
func streamRemoteTunnels(ctx context.Context, p *tea.Program, base string) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, daemonWSURL(base), nil)
	if err != nil {
		return
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var snapshot map[string]monitor.TunnelMetrics
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}
		p.Send(upRemoteStateMsg{connected: true, apiBase: base, snapshot: snapshot})
	}
}
