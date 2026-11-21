package main

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/assets"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Pure view logic of the native tray menu: what each line says, which
// state color (DESIGN.md Four States Rule) its icon gets, the order of the
// VPN list and the aggregate icon badge. Kept free of systray so it is
// tested on every OS.

// stateKind is a menu line's state: its icon color + shape.
type stateKind int

const (
	kindDown       stateKind = iota // grey ring: disconnected
	kindUp                          // green dot: connected
	kindConnecting                  // steel dot: connecting / waiting for the user
	kindCaution                     // caution dot: no answer, dropped
	kindError                       // red dot: failed
)

// tunnelView is one "Conexiones VPN" item.
type tunnelView struct {
	Name    string
	Title   string
	Tooltip string // macOS/Linux only: Windows menus have no item tooltips
	Kind    stateKind
	Checked bool // the tunnel is up: checkmark + listed first
}

// trayStateWord is the localized state word of a tunnel status, the same
// meaning as the CLI's statusText and the dashboards' status labels.
func trayStateWord(s vpn.Status) string {
	switch s.Normalize() {
	case vpn.StatusConnected:
		return trayT("tray.state.connected")
	case vpn.StatusConnecting:
		return trayT("tray.state.connecting")
	case vpn.StatusError:
		return trayT("tray.state.error")
	default:
		return trayT("tray.state.disconnected")
	}
}

// trayTunnelView decides one profile item. Always "name · <state>", never
// a raw status: a phase that needs the user (waiting / dropped) or a
// failed connect replaces the state word with what to do; a server that
// does not answer says so (caution); one that can't be checked keeps the
// state word and adds "sin verificar" (the reason goes to the tooltip and,
// for an outdated helper, to the helper item).
func trayTunnelView(name string, m monitor.TunnelMetrics) tunnelView {
	st := m.Status.Normalize()
	up := st == vpn.StatusConnected
	v := tunnelView{Name: name, Checked: up, Tooltip: trayAttachLine(m.Attach)}
	attach := v.Tooltip
	health := trayHealthLine(m.Health)
	switch {
	case m.Attach != nil && m.Attach.Waiting() && attach != "":
		v.Title = name + " · " + attach
		switch m.Attach.Phase {
		case vpn.PhaseDropped:
			v.Kind = kindCaution
		case vpn.PhaseWaitingDisconnect:
			v.Kind = kindUp
		default:
			v.Kind = kindConnecting
		}
	case m.Health != nil && m.Health.State == vpn.HealthNoResponse:
		v.Title = name + " · " + trayT("tray.state.no_response")
		v.Tooltip, v.Kind = health, kindCaution
	case (m.LastError != "" || m.LastErrorKey != "") && !up:
		v.Title, v.Kind = trayF("tray.tunnel_error", name), kindError
		v.Tooltip = m.LastError
		if m.LastErrorKey != "" {
			v.Tooltip = trayT(m.LastErrorKey)
		}
	default:
		v.Title = name + " · " + trayStateWord(st)
		if health != "" { // HealthUnknown: up, but not verifiable
			v.Title += " · " + trayT("tray.state.unverified")
			v.Tooltip = health
		}
		switch st {
		case vpn.StatusConnected:
			v.Kind = kindUp
		case vpn.StatusConnecting:
			v.Kind = kindConnecting
		case vpn.StatusError:
			v.Kind = kindError
		default:
			v.Kind = kindDown
		}
	}
	return v
}

// trayTunnelViews builds every profile's view, connected ones first, each
// block in name order. A profile the daemon does not report is shown as
// disconnected.
func trayTunnelViews(names []string, states map[string]monitor.TunnelMetrics) []tunnelView {
	views := make([]tunnelView, 0, len(names))
	for _, n := range names {
		views = append(views, trayTunnelView(n, states[n]))
	}
	sort.SliceStable(views, func(i, j int) bool {
		if views[i].Checked != views[j].Checked {
			return views[i].Checked
		}
		return views[i].Name < views[j].Name
	})
	return views
}

// groupView is one "Grupos" item: "name (connected/total)", checked when
// every profile of the group is up.
func groupView(name string, profiles []string, states map[string]monitor.TunnelMetrics) (title string, checked bool) {
	up := 0
	for _, p := range profiles {
		if m, ok := states[p]; ok && m.Status.Normalize() == vpn.StatusConnected {
			up++
		}
	}
	return fmt.Sprintf("%s (%d/%d)", name, up, len(profiles)), len(profiles) > 0 && up == len(profiles)
}

// daemonState is what the tray knows about Kongtrol's background service.
type daemonState int

const (
	daemonRunning      daemonState = iota
	daemonStarting                 // the tray (or a click) is starting it
	daemonStopped                  // nothing listens: `kongtrol down`, quit, crash
	daemonUnresponsive             // it listens but does not answer in time
)

// classifyDaemon maps the result of the status poll. A refused connection
// (nothing listening) is "stopped"; a timeout or an HTTP error from a
// listening daemon is "not responding".
func classifyDaemon(err error, starting bool) daemonState {
	switch {
	case err == nil:
		return daemonRunning
	case starting:
		return daemonStarting
	}
	var ue *url.Error
	if errors.As(err, &ue) && !ue.Timeout() {
		return daemonStopped
	}
	return daemonUnresponsive
}

// trayHeader is the first menu line and the tray icon badge.
func trayHeader(d daemonState, views []tunnelView) (string, assets.TrayBadge) {
	switch d {
	case daemonStopped:
		return trayT("tray.header.stopped"), assets.BadgeStopped
	case daemonStarting:
		return trayT("tray.header.starting"), assets.BadgeNone
	case daemonUnresponsive:
		return trayT("tray.daemon_unreachable"), assets.BadgeCaution
	}
	up, caution, failed := 0, false, false
	for _, v := range views {
		if v.Checked {
			up++
		}
		caution = caution || v.Kind == kindCaution
		failed = failed || v.Kind == kindError
	}
	badge := assets.BadgeNone
	switch {
	case failed:
		badge = assets.BadgeError
	case caution:
		badge = assets.BadgeCaution
	case up > 0:
		badge = assets.BadgeLive
	}
	if up == 0 {
		return trayT("tray.header.none"), badge
	}
	return i18n.N(trayLang, "tray.status_active", up), badge
}

// trayVersionLabel is the "Kongtrol v0.6.0" line: the daemon's version
// (what `kongtrol version` prints), else the tray's own build, else "".
func trayVersionLabel(daemonVersion, trayVersion string) string {
	v := daemonVersion
	if v == "" {
		v = trayVersion
	}
	if v == "" {
		return ""
	}
	if v[0] >= '0' && v[0] <= '9' {
		v = "v" + v
	}
	return trayF("tray.version", strings.TrimSpace(v))
}
