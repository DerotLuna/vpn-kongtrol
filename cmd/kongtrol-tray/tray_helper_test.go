package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func stubHelperProbes(t *testing.T, daemonUp, daemonNeeds, elevated bool, ping func() (privops.PingResult, error)) {
	t.Helper()
	d, e, p := daemonHelperStatus, helperIsElevated, pingHelper
	t.Cleanup(func() { daemonHelperStatus, helperIsElevated, pingHelper = d, e, p })
	daemonHelperStatus = func() (daemonctl.HelperStatus, error) {
		if !daemonUp {
			return daemonctl.HelperStatus{}, errors.New("connection refused")
		}
		return daemonctl.HelperStatus{NeedsInstall: daemonNeeds}, nil
	}
	helperIsElevated = func() bool { return elevated }
	pingHelper = func(context.Context) (privops.PingResult, error) { return ping() }
}

func TestHelperNeedsInstall(t *testing.T) {
	missing := func() (privops.PingResult, error) { return privops.PingResult{}, privops.ErrHelperNotInstalled }
	ok := func() (privops.PingResult, error) {
		return privops.PingResult{Version: "v1", Protocol: privops.ProtocolVersion}, nil
	}
	old := func() (privops.PingResult, error) {
		return privops.PingResult{Version: "v0", Protocol: privops.ProtocolVersion - 1}, nil
	}
	cases := []struct {
		name                   string
		daemonUp, needs, admin bool
		ping                   func() (privops.PingResult, error)
		want                   bool
	}{
		{"daemon says missing", true, true, false, ok, true},
		{"daemon says ready", true, false, false, missing, false},
		// Fresh macOS/Linux install: no daemon yet, no helper.
		{"no daemon, no helper", false, false, false, missing, true},
		{"no daemon, helper answers", false, false, false, ok, false},
		{"no daemon, outdated helper", false, false, false, old, true},
		{"no daemon, elevated", false, false, true, missing, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubHelperProbes(t, c.daemonUp, c.needs, c.admin, c.ping)
			if got := helperNeedsInstall(); got != c.want {
				t.Fatalf("helperNeedsInstall() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClaimOnce(t *testing.T) {
	p := filepath.Join(t.TempDir(), "run", "marker")
	if !claimOnce(p) {
		t.Fatal("first claim must succeed")
	}
	if claimOnce(p) {
		t.Fatal("second claim must not: the notice shows once")
	}
}
