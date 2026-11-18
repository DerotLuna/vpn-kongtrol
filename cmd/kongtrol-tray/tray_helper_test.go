package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
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

func TestHelperState(t *testing.T) {
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
		want                   helperNeed
	}{
		{"daemon says missing", true, true, false, ok, helperMissing},
		{"daemon says ready", true, false, false, missing, helperOK},
		// Fresh macOS/Linux install: no daemon yet, no helper.
		{"no daemon, no helper", false, false, false, missing, helperMissing},
		{"no daemon, helper answers", false, false, false, ok, helperOK},
		{"no daemon, outdated helper", false, false, false, old, helperOutdated},
		{"no daemon, elevated", false, false, true, missing, helperOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubHelperProbes(t, c.daemonUp, c.needs, c.admin, c.ping)
			if got := helperState(); got != c.want {
				t.Fatalf("helperState() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestHelperState_DaemonSaysOutdated(t *testing.T) {
	d := daemonHelperStatus
	t.Cleanup(func() { daemonHelperStatus = d })
	daemonHelperStatus = func() (daemonctl.HelperStatus, error) {
		return daemonctl.HelperStatus{NeedsInstall: true, Outdated: true}, nil
	}
	if got := helperState(); got != helperOutdated {
		t.Fatalf("helperState() = %v, want outdated", got)
	}
	trayLang = i18n.ES
	if title, _ := helperItemText(helperOutdated); title != "Actualizar ayudante (una vez)" {
		t.Fatalf("ES title = %q", title)
	}
	if title, _ := helperItemText(helperMissing); title != "Instalar ayudante (una vez)" {
		t.Fatalf("ES title = %q", title)
	}
	trayLang = i18n.EN
	if title, _ := helperItemText(helperOutdated); title != "Update helper (once)" {
		t.Fatalf("EN title = %q", title)
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
