package main

import (
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// Offers: extra actions a daemon notice carries (monitor.Event.Offer,
// ADR-0102). Clients render each one as a button: the panel, the tray
// flyout, the native tray menu, and the CLI live view (as a command).
const (
	// offerEnableKillSwitch: "Activar protección" turns on the kill
	// switch (POST /api/v1/security/killswitch, kongtrol config killswitch on).
	offerEnableKillSwitch = "enable_killswitch"
	// offerReleaseBlock: "Recuperar internet" disconnects the profile as a
	// user disconnect, so the kill switch lifts its block.
	offerReleaseBlock = "release_block"
	// offerAddProvider: "Agregar <App>" opens the "＋ Agregar" sheet on that
	// provider (Args[len-1] = adapter type).
	offerAddProvider = "add_provider"
)

// blockOffer: every notice about a kill switch block offers "Recuperar
// internet" (the block stays on purpose; a user disconnect lifts it).
func blockOffer(key string) string {
	switch key {
	case "killswitch.event.blocking", "killswitch.event.blocking_full", "killswitch.event.blocking_full_lan",
		"watchdog.event.gave_up_blocking", "watchdog.event.gave_up_blocking_full":
		return offerReleaseBlock
	}
	return ""
}

var (
	killSwitchOfferMu sync.Mutex
	// killSwitchOfferDone: this daemon run already decided (offered, or
	// preferences.json says it was offered before). Keeps the offer to one
	// even when preferences.json cannot be written.
	killSwitchOfferDone bool
	// killSwitchOfferPrefs is the preferences.json path ("" = the user's
	// default); tests point it at a temporary file.
	killSwitchOfferPrefs = ""
	// killSwitchProtects reports whether the kill switch guards a profile.
	killSwitchProtects = func(profile string) bool {
		return killSwitchSvc == nil || killSwitchSvc.ProtectsProfile(profile)
	}
)

// offerKillSwitch runs on a tunnel's first unexpected drop (the watchdog's
// reconnect_attempt: the user did not ask for it). With the kill switch off
// for that profile, its traffic went over the normal internet while
// Kongtrol reconnected it, so the daemon offers to turn the kill switch on,
// once ever: preferences.json remembers it was asked, whatever the answer
// (ADR-0102). "No, gracias" only closes the notice.
func offerKillSwitch(profile string) {
	if killSwitchProtects(profile) {
		return
	}
	killSwitchOfferMu.Lock()
	defer killSwitchOfferMu.Unlock()
	if killSwitchOfferDone {
		return
	}
	killSwitchOfferDone = true
	asked := false
	if _, err := config.UpdatePreferences(killSwitchOfferPrefs, func(p *config.Preferences) error {
		asked = p.KillSwitchOffered
		p.KillSwitchOffered = true
		return nil
	}); err != nil {
		// Offer anyway: once per run is the best that can be kept.
		logAudit("WARN", "killswitch.offer", profile, err.Error())
	}
	if asked {
		return
	}
	daemonEvents.Add(monitor.Event{Kind: monitor.EventTunnelWatchdog, Level: "WARN",
		Key: "killswitch.event.offer", Args: []string{profile}, Profile: profile, Offer: offerEnableKillSwitch})
	msg := cf("killswitch.event.offer", profile)
	emitAlert("WARN", profile, msg)
	logAudit("INFO", "killswitch.offer", profile, msg)
}
