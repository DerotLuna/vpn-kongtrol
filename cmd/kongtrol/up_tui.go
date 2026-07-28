package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
