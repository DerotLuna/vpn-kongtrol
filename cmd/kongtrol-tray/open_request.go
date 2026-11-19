package main

import (
	"errors"
	"log"
	"strings"
	"sync"
)

// Explicit "open this page" request (ADR-0050). The Windows installer's
// finished-page checkbox "Abrir Kongtrol y crear mi primera VPN" starts
// `kongtrol-tray.exe --open /profiles`: the user asked for it, so the page
// opens whatever the one-time first-run marker says, also when profiles
// already exist, and also when a tray is already running (then this second
// process opens it and exits). Only the pages in openAllowed are accepted.

var openAllowed = map[string]bool{"/": true, "/profiles": true}

// parseOpenRequest returns the page asked for with --open <path> or
// --open=<path>, or "" when none (or one not allowed) was given.
func parseOpenRequest(args []string) string {
	for i, a := range args {
		var v string
		switch {
		case a == "--open" && i+1 < len(args):
			v = args[i+1]
		case strings.HasPrefix(a, "--open="):
			v = strings.TrimPrefix(a, "--open=")
		default:
			continue
		}
		if openAllowed[v] {
			return v
		}
		return ""
	}
	return ""
}

var (
	openReqMu sync.Mutex
	openReq   string
)

func setOpenRequest(path string) {
	openReqMu.Lock()
	openReq = path
	openReqMu.Unlock()
}

// takeOpenRequest returns the pending request once.
func takeOpenRequest() string {
	openReqMu.Lock()
	defer openReqMu.Unlock()
	p := openReq
	openReq = ""
	return p
}

// openFromSecondInstance serves --open when another tray already runs:
// start the daemon if needed and open the page, without touching the
// systray (this process never ran systray.Run).
func openFromSecondInstance(path string) {
	if err := initClient(); err != nil && !errors.Is(err, errNoConfig) {
		log.Printf("kongtrol-tray: --open %s: %v", path, err)
		return
	}
	if err := ensureDaemon(); err != nil {
		log.Printf("kongtrol-tray: --open %s: start daemon: %v", path, err)
		return
	}
	panel, _ := client.PanelURL(path)
	if err := openBrowser(panel); err != nil {
		log.Printf("kongtrol-tray: --open %s: browser: %v", path, err)
	}
}
