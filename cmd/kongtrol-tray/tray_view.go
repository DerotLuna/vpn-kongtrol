package main

import (
	"log"
	"sync"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/assets"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// Rendering of the views in tray_state.go onto systray items. systray can
// neither reorder nor insert items after the menu is built, so the VPN list
// is a fixed set of slots that are relabelled on every refresh (connected
// first); each slot's click reads the profile it currently shows.

// version is set at build time: -ldflags "-X main.version=v1.2.3". The
// version line prefers the daemon's own (GET /api/v1/version).
var version = ""

// itemView caches what an item shows, so a refresh only calls systray
// (a Win32 SetMenuItemInfo, a temp icon file) when something changed.
type itemView struct {
	title, tooltip string
	kind           stateKind
	checked        bool
	iconSet        bool
}

// slotItem is one reusable menu item.
type slotItem struct {
	item *systray.MenuItem
	name string // profile or group it shows now
	last itemView
}

func (s *slotItem) render(v itemView, withIcon bool) {
	if s.last.title != v.title {
		s.item.SetTitle(v.title)
	}
	if s.last.tooltip != v.tooltip {
		s.item.SetTooltip(v.tooltip)
	}
	if v.checked != s.last.checked || !s.last.iconSet {
		if v.checked {
			s.item.Check()
		} else {
			s.item.Uncheck()
		}
	}
	if withIcon && (!s.last.iconSet || s.last.kind != v.kind) {
		s.item.SetIcon(stateIcon(v.kind))
	}
	v.iconSet = withIcon || s.last.iconSet
	s.last = v
}

// trayMenu holds the full menu's items.
type trayMenu struct {
	mu       sync.Mutex
	header   *systray.MenuItem
	start    *systray.MenuItem
	tunnels  *systray.MenuItem
	groups   *systray.MenuItem
	dash     *systray.MenuItem
	ver      *systray.MenuItem
	stopAll  *systray.MenuItem
	quit     *systray.MenuItem
	profiles []*slotItem // one per VPN profile, relabelled in view order
	grpSlots []*slotItem // one per group, in name order
	verText  string
	badge    assets.TrayBadge
	theme    string
	iconSet  bool
	daemon   daemonState
}

var stateIconCache sync.Map // stateKind → []byte

// stateIcon is the menu item icon for kind: a dot in the DESIGN.md state
// color on a transparent background; "off" is a hollow ring.
func stateIcon(k stateKind) []byte {
	if b, ok := stateIconCache.Load(k); ok {
		return b.([]byte)
	}
	c, hollow := assets.ColorDim, false
	switch k {
	case kindUp:
		c = assets.ColorLive
	case kindConnecting:
		c = assets.ColorSteel
	case kindCaution:
		c = assets.ColorCaution
	case kindError:
		c = assets.ColorDanger
	default:
		hollow = true
	}
	b := menuIconBytes(assets.StateDotPNG(c, menuIconSize, hollow))
	stateIconCache.Store(k, b)
	return b
}

// setTrayBadge updates the tray icon when the badge or the taskbar theme
// changed (never on every poll: Windows writes a temp .ico per SetIcon).
func (m *trayMenu) setTrayBadge(b assets.TrayBadge) {
	theme := trayIconTheme()
	if m.iconSet && m.badge == b && m.theme == theme {
		return
	}
	if err := applyTrayIcon(b, theme); err != nil {
		log.Printf("kongtrol-tray: tray icon: %v", err)
		return
	}
	m.badge, m.theme, m.iconSet = b, theme, true
}

// renderTunnels relabels the profile slots with views (same length).
func (m *trayMenu) renderTunnels(views []tunnelView) {
	for i, s := range m.profiles {
		if i >= len(views) {
			s.item.Hide()
			continue
		}
		v := views[i]
		m.mu.Lock()
		s.name = v.Name
		m.mu.Unlock()
		s.render(itemView{title: v.Title, tooltip: v.Tooltip, kind: v.Kind, checked: v.Checked}, true)
	}
}

// slotName returns the profile a slot shows now.
func (m *trayMenu) slotName(s *slotItem) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.name
}

// renderGroups updates "name (n/total)" and the checkmark of every group.
func (m *trayMenu) renderGroups(states map[string]monitor.TunnelMetrics) {
	for _, s := range m.grpSlots {
		g, ok := cfg.Groups[s.name]
		if !ok {
			continue
		}
		title, checked := groupView(s.name, g.Profiles, states)
		kind := kindDown
		if checked {
			kind = kindUp
		}
		s.render(itemView{title: title, kind: kind, checked: checked}, false)
	}
}

// renderDaemon shows the header, the "Iniciar Kongtrol" item (only while
// stopped) and the icon badge for the daemon state.
func (m *trayMenu) renderDaemon(d daemonState, header string, badge assets.TrayBadge) {
	m.header.SetTitle(header)
	if d == daemonStopped {
		m.start.Show()
		m.start.Enable()
	} else {
		m.start.Hide()
	}
	if d != daemonRunning {
		m.verText = "" // a restarted daemon may be a newer version
	}
	m.daemon = d
	m.setTrayBadge(badge)
}

// refreshVersion fills the version line once the daemon tells its version
// (or with the tray's own build version until then).
func (m *trayMenu) refreshVersion(running bool) {
	daemonVer := ""
	if running && m.verText == "" {
		if v, err := client.Version(); err == nil {
			daemonVer = v
		}
	}
	if m.verText != "" && daemonVer == "" {
		return
	}
	label := trayVersionLabel(daemonVer, version)
	if label == "" {
		m.ver.Hide()
		return
	}
	m.ver.SetTitle(label)
	m.ver.Show()
	if daemonVer != "" {
		m.verText = label
	}
}
