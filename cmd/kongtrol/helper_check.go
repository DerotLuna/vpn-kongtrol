package main

import (
	"context"
	"errors"
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
