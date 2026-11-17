package main

import (
	"log"
	"sync"
	"time"

	"fyne.io/systray"
)

// noticeTTL is how long a failed action's message stays on the status item
// and icon tooltip before the regular status refresh takes them back.
const noticeTTL = 30 * time.Second

var (
	noticeMu    sync.Mutex
	noticeMsg   string
	noticeUntil time.Time
	// statusItem is the disabled status line of the menu; nil until the
	// full menu is built.
	statusItem *systray.MenuItem
)

// notifyError surfaces a failed tray action to the user: msg (short,
// translated, with a "what to do") goes to the status item and the icon
// tooltip; detail (the raw error) only to tray.log.
func notifyError(msg string, detail error) {
	if detail != nil {
		log.Printf("kongtrol-tray: %s: %v", msg, detail)
	}
	noticeMu.Lock()
	noticeMsg = msg
	noticeUntil = time.Now().Add(noticeTTL)
	item := statusItem
	noticeMu.Unlock()
	if item != nil {
		item.SetTitle(msg)
	}
	systray.SetTooltip(msg)
}

// activeNotice returns the current action error message, or "" once it
// expired.
func activeNotice() string {
	noticeMu.Lock()
	defer noticeMu.Unlock()
	if noticeMsg == "" || time.Now().After(noticeUntil) {
		noticeMsg = ""
		return ""
	}
	return noticeMsg
}
