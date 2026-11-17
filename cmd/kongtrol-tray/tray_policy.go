package main

import (
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Policy notices: the daemon reports traffic of a policy whose tunnel is
// off ("claude.ai sale por internet normal · brainbox está apagado"). The
// tray shows it on the status line and icon tooltip, as a desktop
// notification, and offers a "Conectar brainbox" menu item.

const policyItemTTL = 30 * time.Minute

var (
	policyMu      sync.Mutex
	policyItem    *systray.MenuItem
	policyProfile string
	policyUntil   time.Time
	// policySeq is the last daemon event seen; -1 = first poll, which only
	// learns the current number (no backlog of old notices on tray start).
	policySeq int64 = -1
)

// addPolicyMenuItem creates the hidden "Conectar <perfil>" item.
func addPolicyMenuItem() {
	item := systray.AddMenuItem("", "")
	item.Hide()
	policyMu.Lock()
	policyItem = item
	policyMu.Unlock()
	onClick(item, func() {
		policyMu.Lock()
		name := policyProfile
		policyMu.Unlock()
		if name != "" {
			connectProfile(name)
		}
	})
}

// pollPolicyEvents fetches new daemon notices and updates the item; it
// hides the item once the tunnel is connected or the notice is old.
func pollPolicyEvents(states map[string]monitor.TunnelMetrics) {
	policyMu.Lock()
	after := policySeq
	policyMu.Unlock()
	evs, seq, err := client.Events(after)
	if err != nil {
		return
	}
	policyMu.Lock()
	policySeq = seq
	policyMu.Unlock()
	for _, e := range evs {
		if e.Kind != monitor.EventPolicyTunnelDown && e.Kind != monitor.EventTunnelWatchdog {
			continue
		}
		args := make([]any, len(e.Args))
		for i, a := range e.Args {
			args[i] = a
		}
		msg := trayF(e.Key, args...)
		notifyError(msg, nil)
		desktopNotify("VPN Kongtrol", msg)
		if e.CanConnect && e.Profile != "" {
			policyMu.Lock()
			policyProfile = e.Profile
			policyUntil = time.Now().Add(policyItemTTL)
			item := policyItem
			policyMu.Unlock()
			if item != nil {
				item.SetTitle(trayF("policy.event.action_connect", e.Profile))
				item.SetTooltip(msg)
				item.Show()
			}
		}
	}
	policyMu.Lock()
	defer policyMu.Unlock()
	if policyItem == nil || policyProfile == "" {
		return
	}
	if m, ok := states[policyProfile]; (ok && m.Status == vpn.StatusConnected) || time.Now().After(policyUntil) {
		policyItem.Hide()
		policyProfile = ""
	}
}

var ksItem *systray.MenuItem

// addKillSwitchMenuItem creates the disabled kill switch state line.
func addKillSwitchMenuItem() {
	item := systray.AddMenuItem("", "")
	item.Disable()
	item.Hide()
	policyMu.Lock()
	ksItem = item
	policyMu.Unlock()
}

// refreshKillSwitchItem shows "Kill switch: ARMADO (protegiendo 2 túneles)"
// / "ACTIVO (bloqueando destinos de brainbox)"; hidden when off.
func refreshKillSwitchItem() {
	policyMu.Lock()
	item := ksItem
	policyMu.Unlock()
	if item == nil {
		return
	}
	st, err := client.KillSwitch()
	if err != nil || st == nil || st.State == "off" {
		item.Hide()
		return
	}
	key, args := st.StateKey()
	item.SetTitle(trayF("tray.kill_switch", trayF(key, args...)))
	item.Show()
}
