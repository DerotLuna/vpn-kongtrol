package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// helperNeed says whether the privileged helper has to be installed.
type helperNeed int

const (
	helperReady    helperNeed = iota // answers with this protocol, or not needed (elevated)
	helperMissing                    // not installed, not running or refuses this user
	helperOutdated                   // answers with another protocol version
)

// guidanceKey is the translated "what to do" line for n.
func (n helperNeed) guidanceKey() string {
	if n == helperOutdated {
		return "service.error.outdated"
	}
	return "service.error.not_installed"
}

// Test seams.
var (
	helperStateFn   = helperState
	installHelperFn = func() error { _, err := installHelper(); return err }
)

// helperState pings the helper the same way the daemon decides
// needs_install (GET /api/v1/helper/status): an elevated process needs no
// helper; otherwise it must answer with privops.ProtocolVersion.
func helperState() helperNeed {
	if privops.IsElevated() {
		return helperReady
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := privops.NewClient().Ping(ctx)
	switch {
	case errors.Is(err, privops.ErrHelperOutdated):
		return helperOutdated
	case err != nil:
		return helperMissing
	case res.Protocol != privops.ProtocolVersion:
		return helperOutdated
	}
	return helperReady
}

// offerHelperInstall is the first-run check shared by init and up: when the
// helper is missing or outdated it says why Kongtrol needs it and offers to
// install it now (default yes; yes skips the question). Without an
// interactive terminal it only prints the "what to do" line. It reports
// whether the helper answers afterwards.
func offerHelperInstall(need helperNeed, yes bool) bool {
	if need == helperReady {
		return true
	}
	if need == helperOutdated {
		fmt.Println(tuiWarn(ct("cli.helper.outdated")))
	} else {
		fmt.Println(tuiWarn(ct("cli.helper.missing")))
	}
	fmt.Println(tuiInfo(paint(cDim, ct("cli.helper.why"))))
	if !confirmAction(ct("cli.helper.ask_install"), yes, true) {
		fmt.Println(tuiInfo(ct(need.guidanceKey())))
		return false
	}
	if err := installHelperFn(); err != nil {
		fmt.Println(tuiWarn(err.Error()))
		return false
	}
	fmt.Println(tuiOK(ct("service.install.ok")))
	return true
}

// helperNeedFromKey maps a guidance key the daemon sent back (a tunnel's
// last_error_key or an API error_key) to what the helper needs.
func helperNeedFromKey(key string) helperNeed {
	switch key {
	case "service.error.not_installed":
		return helperMissing
	case "service.error.outdated":
		return helperOutdated
	}
	return helperReady
}
