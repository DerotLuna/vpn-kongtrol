package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// ADR-0101: provider apps are never launched at start.
func TestPlanRestore(t *testing.T) {
	types := map[string]string{"home": "wireguard", "proton": "protonvpn", "nord": "nordvpn", "work": "openvpn"}
	appUp := map[string]bool{"protonvpn": true}
	p := planRestore([]string{"home", "proton", "nord", "work"},
		func(n string) string { return types[n] },
		func(typ string) (bool, bool) {
			provider := typ == "protonvpn" || typ == "nordvpn"
			return appUp[typ], provider
		})
	if !slices.Equal(p.direct, []string{"home", "work"}) || !slices.Equal(p.attach, []string{"proton"}) || !slices.Equal(p.skipped, []string{"nord"}) {
		t.Fatalf("plan = %+v", p)
	}
}

// fakeClock counts the waits; every wait returns at once.
type fakeClock struct{ waits []time.Duration }

func (c *fakeClock) after(d time.Duration) <-chan time.Time {
	c.waits = append(c.waits, d)
	ch := make(chan time.Time, 1)
	ch <- time.Time{}
	return ch
}

type restoreRecorder struct {
	connected []string
	said      []string
}

func (r *restoreRecorder) deps(ready func() bool, clock *fakeClock, netChanged func(context.Context) bool) restoreDeps {
	return restoreDeps{
		typeOf: func(n string) string {
			if n == "proton" {
				return "protonvpn"
			}
			return "wireguard"
		},
		appTunnel: func(typ string) (bool, bool) { return false, typ == "protonvpn" },
		wait: monitor.NetworkWait{Ready: ready, Interval: monitor.NetworkWaitInterval,
			Timeout: monitor.NetworkWaitTimeout, After: clock.after},
		netChanged: netChanged,
		stillWant:  func(string) bool { return true },
		connect:    func(n string) { r.connected = append(r.connected, n) },
		say:        func(_ string, key string, _ ...any) { r.said = append(r.said, key) },
	}
}

func TestRunRestore_WaitsForNetwork(t *testing.T) {
	calls := 0
	ready := func() bool { calls++; return calls > 3 } // up after a few polls
	clock := &fakeClock{}
	rec := &restoreRecorder{}
	never := func(context.Context) bool { t.Fatal("network change awaited although the wait succeeded"); return false }
	runRestore(context.Background(), []string{"home", "proton"}, rec.deps(ready, clock, never))

	if !slices.Equal(rec.connected, []string{"home"}) {
		t.Fatalf("connected %v: the provider app must not be launched", rec.connected)
	}
	if !slices.Equal(rec.said, []string{"cli.daemon.restore_app_skipped", "cli.daemon.restore_waiting_network"}) {
		t.Fatalf("said %v", rec.said)
	}
	if len(clock.waits) == 0 || clock.waits[0] != 2*time.Second {
		t.Fatalf("waits %v, want 2 s polls", clock.waits)
	}
}

func TestRunRestore_ReadyNetworkSaysNothing(t *testing.T) {
	rec := &restoreRecorder{}
	clock := &fakeClock{}
	runRestore(context.Background(), []string{"home"}, rec.deps(func() bool { return true }, clock, nil))
	if !slices.Equal(rec.connected, []string{"home"}) || len(rec.said) != 0 || len(clock.waits) != 0 {
		t.Fatalf("connected %v said %v waits %v", rec.connected, rec.said, clock.waits)
	}
}

// No network after 90 s: no notice, the next network change restores.
func TestRunRestore_NoNetworkWaitsForChange(t *testing.T) {
	clock := &fakeClock{}
	rec := &restoreRecorder{}
	changed := 0
	runRestore(context.Background(), []string{"home"}, rec.deps(func() bool { return false }, clock,
		func(context.Context) bool { changed++; return true }))
	if got := len(clock.waits); got != 45 {
		t.Errorf("polls = %d, want 45 (90 s every 2 s)", got)
	}
	if changed != 1 || !slices.Equal(rec.connected, []string{"home"}) {
		t.Fatalf("changed %d connected %v", changed, rec.connected)
	}
	if !slices.Equal(rec.said, []string{"cli.daemon.restore_waiting_network", "cli.daemon.restore_no_network"}) {
		t.Fatalf("said %v", rec.said)
	}

	// The daemon stops while waiting for the change: nothing connects.
	rec = &restoreRecorder{}
	runRestore(context.Background(), []string{"home"}, rec.deps(func() bool { return false }, &fakeClock{},
		func(context.Context) bool { return false }))
	if len(rec.connected) != 0 {
		t.Fatalf("connected %v without network", rec.connected)
	}
}

// A profile the user turned off (or restore turned off) while waiting is
// not connected.
func TestRunRestore_SkipsNoLongerWanted(t *testing.T) {
	rec := &restoreRecorder{}
	d := rec.deps(func() bool { return true }, &fakeClock{}, nil)
	d.stillWant = func(n string) bool { return n != "home" }
	runRestore(context.Background(), []string{"home", "work"}, d)
	if !slices.Equal(rec.connected, []string{"work"}) {
		t.Fatalf("connected %v", rec.connected)
	}
}

// The first successful user connect makes the question pending, once;
// an explicit choice is never asked again.
func TestMarkRestoreQuestion(t *testing.T) {
	old, oldFn := restorePrefsPath, restoreQuestionChanged
	t.Cleanup(func() { restorePrefsPath, restoreQuestionChanged = old, oldFn })
	restorePrefsPath = filepath.Join(t.TempDir(), "preferences.json")
	announced := 0
	restoreQuestionChanged = func() { announced++ }

	markRestoreQuestion()
	markRestoreQuestion()
	prefs, _ := config.LoadPreferences(restorePrefsPath)
	if !prefs.RestoreQuestionPending() || prefs.RestoreTunnelsEnabled() || announced != 1 {
		t.Fatalf("after first connect: %+v announced=%d", prefs, announced)
	}

	// An explicit choice (here "Ahora no") is kept and not asked again.
	_, _ = config.UpdatePreferences(restorePrefsPath, func(p *config.Preferences) error { p.SetRestoreTunnels(false); return nil })
	markRestoreQuestion()
	prefs, _ = config.LoadPreferences(restorePrefsPath)
	if prefs.RestoreQuestionPending() || prefs.RestoreAsk || announced != 1 {
		t.Fatalf("after answer: %+v announced=%d", prefs, announced)
	}
}

type fakeAnswerer struct{ answers []bool }

func (f *fakeAnswerer) AnswerRestoreQuestion(on bool) error {
	f.answers = append(f.answers, on)
	return nil
}

func withHomePrefs(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KONGTROL_LANG", "es")
	return filepath.Join(home, ".kongtrol", "preferences.json")
}

func TestOfferRestoreAtStart(t *testing.T) {
	withHomePrefs(t)
	oldPrompt, oldJSON, oldQuiet := promptInteractiveFn, outputJSON, outputQuiet
	t.Cleanup(func() {
		promptInteractiveFn, outputJSON, outputQuiet = oldPrompt, oldJSON, oldQuiet
		promptReader = os.Stdin
	})

	// No terminal: one "how to" line, no answer saved.
	promptInteractiveFn = func() bool { return false }
	f := &fakeAnswerer{}
	out, _ := captureStdout(t, func() error { offerRestoreAtStart(f); return nil })
	if len(f.answers) != 0 || !strings.Contains(out, "kongtrol config restore-tunnels on") {
		t.Fatalf("non-interactive: %v\n%s", f.answers, out)
	}

	// --json: silent.
	outputJSON = true
	out, _ = captureStdout(t, func() error { offerRestoreAtStart(f); return nil })
	if out != "" || len(f.answers) != 0 {
		t.Fatalf("--json printed %q", out)
	}
	outputJSON = false

	// A terminal: Enter takes the default (yes).
	promptInteractiveFn = func() bool { return true }
	promptReader = strings.NewReader("\n")
	out, _ = captureStdout(t, func() error { offerRestoreAtStart(f); return nil })
	if !slices.Equal(f.answers, []bool{true}) || !strings.Contains(out, "¿Volver a conectar tus VPN al encender el equipo?") {
		t.Fatalf("interactive: %v\n%s", f.answers, out)
	}

	// Already answered: never asked again.
	_, _ = config.UpdatePreferences("", func(p *config.Preferences) error { p.SetRestoreTunnels(false); return nil })
	promptReader = strings.NewReader("s\n")
	_, _ = captureStdout(t, func() error { offerRestoreAtStart(f); return nil })
	if len(f.answers) != 1 {
		t.Fatalf("asked again: %v", f.answers)
	}
}
