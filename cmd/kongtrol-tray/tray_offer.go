package main

import (
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Notice offers in the native menu (ADR-0102): a daemon notice may carry
// one extra action (monitor.Event.Offer). The tray shows it as one menu
// item next to "Conectar <perfil>": "Activar protección" (kill switch on),
// "Recuperar internet" (disconnect the blocked profile, as the user), or
// "Agregar <App>" (the panel's "＋ Agregar" sheet on that provider).

const (
	offerEnableKillSwitch = "enable_killswitch"
	offerReleaseBlock     = "release_block"
	offerAddProvider      = "add_provider"
)

// trayOffer is one offer the menu can show.
type trayOffer struct {
	kind     string
	profile  string
	provider string // adapter type, add_provider only
}

// offerOf reads the offer of a daemon notice; false when it has none the
// tray knows (an unknown offer from a newer daemon is ignored).
func offerOf(e monitor.Event) (trayOffer, bool) {
	o := trayOffer{kind: e.Offer, profile: e.Profile}
	switch e.Offer {
	case offerEnableKillSwitch:
		return o, true
	case offerReleaseBlock:
		if e.Profile == "" {
			return trayOffer{}, false
		}
		return o, true
	case offerAddProvider:
		if len(e.Args) == 0 || addProviderPath(e.Args[len(e.Args)-1]) == "" {
			return trayOffer{}, false
		}
		o.provider = e.Args[len(e.Args)-1]
		return o, true
	}
	return trayOffer{}, false
}

// title is the menu item text in the tray language.
func (o trayOffer) title() string {
	switch o.kind {
	case offerEnableKillSwitch:
		return trayT("offer.action.enable_killswitch")
	case offerReleaseBlock:
		return trayT("offer.action.release_block") + " · " + o.profile
	default:
		return trayF("offer.action.add_provider", i18n.OfferAppName(o.provider))
	}
}

// run does what the offer says.
func (o trayOffer) run() {
	switch o.kind {
	case offerEnableKillSwitch:
		if err := client.SetKillSwitch(true); err != nil {
			notifyError(trayT("offer.err.enable_killswitch"), err)
		}
	case offerReleaseBlock:
		disconnectProfile(o.profile) // a user disconnect: the block lifts
	case offerAddProvider:
		openDashboardPage(addProviderPath(o.provider))
	}
}

var (
	offerItem  *systray.MenuItem
	offerShown trayOffer
	offerUntil time.Time
	offerMu    sync.Mutex
)

// addOfferMenuItem creates the hidden offer item.
func addOfferMenuItem() {
	item := systray.AddMenuItem("", "")
	item.Hide()
	offerMu.Lock()
	offerItem = item
	offerMu.Unlock()
	onClick(item, func() {
		offerMu.Lock()
		o := offerShown
		offerShown = trayOffer{}
		offerMu.Unlock()
		item.Hide()
		if o.kind != "" {
			go o.run()
		}
	})
}

// showOffer puts the notice's offer in the menu (the latest one wins).
func showOffer(e monitor.Event, tooltip string) {
	o, ok := offerOf(e)
	if !ok {
		return
	}
	offerMu.Lock()
	defer offerMu.Unlock()
	if offerItem == nil {
		return
	}
	offerShown = o
	offerUntil = time.Now().Add(policyItemTTL)
	offerItem.SetTitle(o.title())
	offerItem.SetTooltip(tooltip)
	offerItem.Show()
}

// expireOffer hides the offer once it is old, or once the profile it is
// about is connected again (nothing left to release or protect now).
func expireOffer(states map[string]monitor.TunnelMetrics) {
	offerMu.Lock()
	defer offerMu.Unlock()
	if offerItem == nil || offerShown.kind == "" {
		return
	}
	m, ok := states[offerShown.profile]
	back := offerShown.kind == offerReleaseBlock && ok && m.Status == vpn.StatusConnected
	if back || time.Now().After(offerUntil) {
		offerItem.Hide()
		offerShown = trayOffer{}
	}
}
