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
)

// Restore at start (ADR-0098). wantedTunnels is the persisted set of
// profiles the user wants up; nil outside the daemon (tests, CLI), where
// every use below does nothing.
var wantedTunnels *daemonctl.Wanted

// userConnectProfile is connectProfile for a request from the user (panel,
// tray, `kongtrol up`, groups): a profile that comes up is remembered, so
// it is reconnected when Kongtrol starts again. The watchdog, the
// scheduler and rule auto-connect use connectProfile directly.
func userConnectProfile(ctx context.Context, name string) error {
	err := connectProfile(ctx, name)
	if err == nil {
		_ = wantedTunnels.Add(name)
	}
	return err
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

// restoreWanted reconnects, in the background, the profiles the user had
// on when "Al iniciar, volver a conectar" is on. Each gets one connect now
// (a provider app attaches, or waits for the user, as with `kongtrol up`);
// when it fails, the watchdog makes the remaining attempts of the same
// bounded policy and then tells the user once (ADR-0098).
func restoreWanted(ctx context.Context, already []string) {
	if wantedTunnels == nil || setupMode {
		return
	}
	exists := func(name string) bool { _, ok := adapters.Get(name); return ok }
	saved, err := wantedTunnels.Keep(exists)
	if err != nil {
		saved = wantedTunnels.List()
	}
	prefs, err := loadPreferences()
	if err != nil || !prefs.RestoreTunnelsEnabled() {
		return
	}
	targets := restoreTargets(saved, already, exists)
	if len(targets) == 0 {
		return
	}
	fmt.Println(tuiInfo(cf("cli.daemon.restoring", strings.Join(targets, ", "))))
	logAudit("INFO", "vpn.restore", "", cf("cli.daemon.restoring", strings.Join(targets, ", ")))
	for _, name := range targets {
		go func() {
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
		}()
	}
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
