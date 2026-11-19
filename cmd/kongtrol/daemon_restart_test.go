package main

import (
	"slices"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestReplacementArgs(t *testing.T) {
	oldPath, oldBell := activeCfgPath, alertBell
	t.Cleanup(func() { activeCfgPath, alertBell = oldPath, oldBell })
	activeCfgPath, alertBell = "", false

	got := replacementArgs([]string{"office"})
	want := []string{"daemon", "--plain", "--restart", "--", "office"}
	if !slices.Equal(got, want) {
		t.Fatalf("args=%v, want %v", got, want)
	}
}

// A profile name that looks like a flag stays a name (after "--"); one the
// config validator refuses never reaches the command line.
func TestReplacementArgs_NamesAreNeverFlags(t *testing.T) {
	oldPath, oldBell := activeCfgPath, alertBell
	t.Cleanup(func() { activeCfgPath, alertBell = oldPath, oldBell })
	activeCfgPath, alertBell = "", false

	got := replacementArgs([]string{"--config=/tmp/evil.yaml", "bad\nname", "office"})
	want := []string{"daemon", "--plain", "--restart", "--", "--config=/tmp/evil.yaml", "office"}
	if !slices.Equal(got, want) {
		t.Fatalf("args=%v, want %v", got, want)
	}
	// The same flag parser cobra uses, with the daemon's flags.
	fs := pflag.NewFlagSet("daemon", pflag.ContinueOnError)
	fs.Bool("plain", false, "")
	fs.Bool("restart", false, "")
	cfgFlag := fs.String("config", "", "")
	if err := fs.Parse(got[1:]); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pos := fs.Args(); !slices.Equal(pos, []string{"--config=/tmp/evil.yaml", "office"}) {
		t.Fatalf("positional=%v", pos)
	}
	if *cfgFlag != "" {
		t.Fatal("a profile name became the --config flag")
	}
}

func TestKnownTargets_DropsDeletedProfiles(t *testing.T) {
	old := adapters
	t.Cleanup(func() { adapters = old })
	adapters = vpn.NewSet(map[string]vpn.VPNAdapter{"office": nil})

	got := knownTargets([]string{"office", "gone"})
	if !slices.Equal(got, []string{"office"}) {
		t.Fatalf("targets=%v, want [office]", got)
	}
}

func TestWaitDaemonGone(t *testing.T) {
	oldProbe, oldSleep := daemonProbeFn, restartSleep
	t.Cleanup(func() { daemonProbeFn, restartSleep = oldProbe, oldSleep })
	restartSleep = func(time.Duration) {}

	answers := []bool{true, true, false}
	daemonProbeFn = func(string) bool {
		v := answers[0]
		if len(answers) > 1 {
			answers = answers[1:]
		}
		return v
	}
	if !waitDaemonGone("http://x", time.Minute) {
		t.Fatal("waitDaemonGone=false, want true once the probe fails")
	}

	daemonProbeFn = func(string) bool { return true }
	if waitDaemonGone("http://x", 0) {
		t.Fatal("waitDaemonGone=true for a daemon that never stops")
	}
}
