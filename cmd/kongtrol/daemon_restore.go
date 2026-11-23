package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/autostart"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// Restore at start (ADR-0098). wantedTunnels is the persisted set of
// profiles the user wants up; nil outside the daemon (tests, CLI), where
// every use below does nothing.
var wantedTunnels *daemonctl.Wanted

// restorePrefsPath is the preferences.json the daemon reads for the
// restore choice ("" = the user's default). A seam for tests.
var restorePrefsPath = ""

// restoreQuestionChanged wakes the panel and the tray flyout when the
// restore question becomes pending (api.Server.AnnounceChange).
var restoreQuestionChanged func()

// userConnectProfile is connectProfile for a request from the user (panel,
// tray, `kongtrol up`, groups): a profile that comes up is remembered, so
// it is reconnected when Kongtrol starts again. The first one also makes
// the restore question pending (ADR-0101). The watchdog, the scheduler
// and rule auto-connect use connectProfile directly.
func userConnectProfile(ctx context.Context, name string) error {
	err := connectProfile(ctx, name)
	if err == nil && wantedTunnels != nil {
		_ = wantedTunnels.Add(name)
		markRestoreQuestion()
	}
	return err
}

// markRestoreQuestion makes "¿Volver a conectar tus VPN al encender el
// equipo?" pending unless the user already chose (ADR-0101). The file is
// written only once: when the question becomes pending.
func markRestoreQuestion() {
	prefs, err := config.LoadPreferences(restorePrefsPath)
	if err != nil || prefs.RestoreAnswered() || prefs.RestoreAsk {
		return
	}
	changed := false
	if _, err := config.UpdatePreferences(restorePrefsPath, func(p *config.Preferences) error {
		changed = p.MarkRestoreQuestion()
		return nil
	}); err != nil || !changed {
		return
	}
	if restoreQuestionChanged != nil {
		restoreQuestionChanged()
	}
}

// userDisconnectProfile is disconnectProfile for a request from the user:
// the profile is forgotten first, so even a disconnect that has to wait
// for the provider's app is never restored at the next start.
func userDisconnectProfile(ctx context.Context, name string) error {
	_ = wantedTunnels.Remove(name)
	return disconnectProfile(ctx, name)
}

// forgetWantedTunnels empties the set: the user stopped everything
// (`kongtrol down`, "Salir y desconectar todo") or deleted all data.
func forgetWantedTunnels() { _ = wantedTunnels.Clear() }

// restoreTargets returns the saved profiles to reconnect at start: those
// that still exist and that this start does not connect already.
func restoreTargets(saved, already []string, exists func(string) bool) []string {
	var out []string
	for _, name := range saved {
		if exists(name) && !slices.Contains(already, name) {
			out = append(out, name)
		}
	}
	return out
}

// restorePlan splits the profiles to restore (ADR-0101). A provider-app
// profile (Proton, NordVPN, Surfshark, FortiClient: any adapter type with
// an appattach matcher) is never launched at start: it is attached when
// its app already has the tunnel up, and skipped otherwise (it stays in
// wanted.json). Direct profiles wait for the network first.
type restorePlan struct {
	direct  []string
	attach  []string
	skipped []string
}

// planRestore decides with typeOf (profile → adapter type) and appTunnel
// (appattach.ActiveTunnel: up now, and whether typ is a provider app).
func planRestore(targets []string, typeOf func(string) string, appTunnel func(typ string) (up, known bool)) restorePlan {
	var p restorePlan
	for _, name := range targets {
		up, provider := appTunnel(typeOf(name))
		switch {
		case !provider:
			p.direct = append(p.direct, name)
		case up:
			p.attach = append(p.attach, name)
		default:
			p.skipped = append(p.skipped, name)
		}
	}
	return p
}

// restoreDeps are what runRestore needs from the daemon; tests replace
// them.
type restoreDeps struct {
	typeOf     func(string) string
	appTunnel  func(typ string) (up, known bool)
	wait       monitor.NetworkWait
	netChanged func(ctx context.Context) bool // after a failed wait
	stillWant  func(string) bool              // still wanted and restore on
	connect    func(name string)
	say        func(level, key string, args ...any)
}

// runRestore attaches provider tunnels already up, skips provider apps
// that are not, waits for the network (2 s polls, up to 90 s; no
// watchdog attempt is spent), and then connects the direct profiles.
// With no network after the wait it says so once and waits for the next
// network change, like the watchdog's re-arm (ADR-0098, ADR-0101).
func runRestore(ctx context.Context, targets []string, d restoreDeps) {
	plan := planRestore(targets, d.typeOf, d.appTunnel)
	for _, name := range plan.skipped {
		d.say("INFO", "cli.daemon.restore_app_skipped", name)
	}
	for _, name := range plan.attach {
		d.connect(name)
	}
	if len(plan.direct) == 0 {
		return
	}
	list := strings.Join(plan.direct, ", ")
	if !d.wait.Ready() {
		d.say("INFO", "cli.daemon.restore_waiting_network", list)
		if !d.wait.Wait(ctx) {
			if ctx.Err() != nil {
				return
			}
			d.say("INFO", "cli.daemon.restore_no_network", list)
			if !d.netChanged(ctx) {
				return
			}
		}
	}
	for _, name := range plan.direct {
		if ctx.Err() != nil {
			return
		}
		if d.stillWant(name) {
			d.connect(name)
		}
	}
}

// restoreWanted reconnects, in the background, the profiles the user had
// on when "Al iniciar, volver a conectar" is on. Each gets one connect
// once the network is there (a provider app only attaches); when it
// fails, the watchdog makes the remaining attempts of the same bounded
// policy and then tells the user once (ADR-0098, ADR-0101).
func restoreWanted(ctx context.Context, already []string) {
	if wantedTunnels == nil || setupMode {
		return
	}
	exists := func(name string) bool { _, ok := adapters.Get(name); return ok }
	saved, err := wantedTunnels.Keep(exists)
	if err != nil {
		saved = wantedTunnels.List()
	}
	if !restoreEnabled() {
		return
	}
	targets := restoreTargets(saved, already, exists)
	if len(targets) == 0 {
		return
	}
	fmt.Println(tuiInfo(cf("cli.daemon.restoring", strings.Join(targets, ", "))))
	logAudit("INFO", "vpn.restore", "", cf("cli.daemon.restoring", strings.Join(targets, ", ")))
	ready := func() bool { return monitor.NetworkReady(ownTunnelInterface) }
	go runRestore(ctx, targets, restoreDeps{
		typeOf: profileType,
		appTunnel: func(typ string) (bool, bool) {
			_, up, known := appattach.ActiveTunnel(typ)
			return up, known
		},
		wait: monitor.NetworkWait{Ready: ready, Interval: monitor.NetworkWaitInterval, Timeout: monitor.NetworkWaitTimeout},
		netChanged: func(ctx context.Context) bool {
			return monitor.WaitNetworkChange(ctx, monitor.NetworkFingerprint, ready, 5*time.Second, nil)
		},
		stillWant: func(name string) bool {
			return restoreEnabled() && slices.Contains(wantedTunnels.List(), name)
		},
		connect: func(name string) { go restoreConnect(ctx, name) },
		say: func(level, key string, args ...any) {
			msg := cf(key, args...)
			fmt.Println(tuiInfo(msg))
			logAudit(level, "vpn.restore", "", msg)
		},
	})
}

// restoreEnabled reads the restore choice now (the user may change it
// while the daemon waits for the network).
func restoreEnabled() bool {
	prefs, err := config.LoadPreferences(restorePrefsPath)
	return err == nil && prefs.RestoreTunnelsEnabled()
}

// restoreConnect makes the one connect of a restored profile.
func restoreConnect(ctx context.Context, name string) {
	started := time.Now()
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	err := connectProfile(connectCtx, name)
	cancel()
	switch {
	case err == nil:
		fmt.Println(tuiOK(name + "  " + cf("cli.daemon.up.connected_in", time.Since(started).Round(time.Second))))
	case ctx.Err() != nil || errors.Is(err, context.Canceled):
		// The daemon stops, or the user turned it off meanwhile.
	case !slices.Contains(wantedTunnels.List(), name):
		// Turned off by the user while it connected.
	default:
		fmt.Println(tuiWarn(name + "  " + err.Error()))
		if watchdog != nil {
			watchdog.MarkActiveAfterFailedConnect(name)
		}
	}
}

// profileType is the adapter type of a profile in the loaded config.
func profileType(name string) string {
	if c := cfg; c != nil {
		return c.VPNs[name].Type
	}
	return ""
}

// ownTunnelInterface reports whether iface is the interface of a tunnel
// Kongtrol itself brings up (WireGuard, OpenVPN…): a default route through
// it is not "the network is there". A provider app's tunnel counts as
// network, like any other connection the user has.
func ownTunnelInterface(iface string) bool {
	if col == nil {
		return false
	}
	for name, m := range col.Snapshot() {
		if m.InterfaceName != iface {
			continue
		}
		if _, _, provider := appattach.ActiveTunnel(profileType(name)); !provider {
			return true
		}
	}
	return false
}

// pruneWanted drops profiles a saved config no longer has.
func pruneWanted(c *config.Config) {
	if wantedTunnels == nil || c == nil {
		return
	}
	_, _ = wantedTunnels.Keep(func(name string) bool { _, ok := c.VPNs[name]; return ok })
}

// startupHooks wires /api/v1/startup for the daemon.
func startupHooks() api.StartupHooks {
	return api.StartupHooks{
		Autostart:    autostart.Status,
		SetAutostart: autostart.Set,
		OnStopAll:    forgetWantedTunnels,
	}
}
