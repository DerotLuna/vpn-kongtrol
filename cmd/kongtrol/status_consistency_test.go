package main

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// fakeKS is a local kill switch that never engages, like the viewer
// process's own instance while the daemon owns the real one.
type fakeKS struct{ on bool }

func (f *fakeKS) Enable(string, bool) error { f.on = true; return nil }
func (f *fakeKS) Disable() error            { f.on = false; return nil }
func (f *fakeKS) IsEnabled() bool           { return f.on }

func TestKillSwitchStateKey(t *testing.T) {
	cases := []struct {
		enabled, active bool
		want            string
	}{
		{false, false, "cli.status.security.off_caps"},
		{true, false, "cli.status.security.armed"},
		{true, true, "cli.status.security.active_caps"},
		// Rules in place win even if the setting was just turned off.
		{false, true, "cli.status.security.active_caps"},
	}
	for _, c := range cases {
		if got := killSwitchStateKey(c.enabled, c.active); got != c.want {
			t.Errorf("killSwitchStateKey(%v,%v)=%q want %q", c.enabled, c.active, got, c.want)
		}
	}
}

// The live view must report the daemon's kill switch state, not the
// viewer's own never-engaged instance — same source as `kongtrol status`.
func TestUpModelSecurityStateFromDaemon(t *testing.T) {
	m := upModel{ks: &fakeKS{}, ksOn: true}
	if _, active, _, _ := m.securityState(); active {
		t.Fatal("local fallback: want inactive")
	}
	m.remoteSec = &apiSecurityStatus{KillSwitchEnabled: true, KillSwitch: true, DNSGuardEnabled: true, DNSGuard: true}
	ksOn, ksActive, dnsOn, dnsActive := m.securityState()
	if !ksOn || !ksActive || !dnsOn || !dnsActive {
		t.Fatalf("daemon state ignored: %v %v %v %v", ksOn, ksActive, dnsOn, dnsActive)
	}
	next, _ := m.Update(upRemoteStateMsg{connected: false})
	if next.(upModel).remoteSec != nil {
		t.Fatal("remote security must be dropped when the daemon goes away")
	}
}

func TestIsInternalCommand(t *testing.T) {
	root := &cobra.Command{Use: "kongtrol"}
	daemon := &cobra.Command{Use: "daemon", Hidden: true}
	helper := &cobra.Command{Use: "helper", Hidden: true}
	run := &cobra.Command{Use: "run"}
	status := &cobra.Command{Use: "status"}
	root.AddCommand(daemon, helper, status)
	helper.AddCommand(run)
	for _, c := range []*cobra.Command{daemon, run} {
		if !isInternalCommand(c) {
			t.Errorf("%s should be internal", c.CommandPath())
		}
	}
	if isInternalCommand(status) {
		t.Error("status is user-facing")
	}
}

func TestFormatStatusUptimeLongUsesDate(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.Local)
	old := now.Add(-2633 * time.Hour)
	got := formatStatusUptime(now, vpn.StatusDisconnected, time.Time{}, monitor.ProfileHistory{LastDownAt: old})
	if strings.Contains(got, "2633h") || !strings.Contains(got, formatLocalDate(old)) {
		t.Fatalf("long down: %q", got)
	}
	short := formatStatusUptime(now, vpn.StatusDisconnected, time.Time{}, monitor.ProfileHistory{LastDownAt: now.Add(-90 * time.Minute)})
	if !strings.Contains(short, "1h 30m") {
		t.Fatalf("short down should keep elapsed time: %q", short)
	}
	up := formatStatusUptime(now, vpn.StatusConnected, now.Add(-72*time.Hour), monitor.ProfileHistory{})
	if !strings.Contains(up, formatLocalDate(now.Add(-72*time.Hour))) {
		t.Fatalf("long up: %q", up)
	}
}

// doctor gets the daemon's own state (it owns tunnels, kill switch and DNS
// guard); internal/doctor tests that the state wins over fresh adapters.
func TestDoctorDaemonFromSnapshot(t *testing.T) {
	d := doctorDaemonFromSnapshot([]monitor.TunnelMetrics{{Name: "a", Status: vpn.StatusConnected}},
		&apiSecurityStatus{DNSGuardEnabled: true, DNSGuard: true, KillSwitchEnabled: true})
	if !d.DNSGuardActive || !d.KillSwitchEnabled {
		t.Fatalf("security state lost: %+v", d)
	}
	if d.Tunnels["a"] != vpn.StatusConnected {
		t.Fatal("daemon tunnel state missing")
	}
}
