package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// failReader fails the test if anything tries to read an answer.
type failReader struct{ t *testing.T }

func (r failReader) Read([]byte) (int, error) {
	r.t.Fatal("must not wait for input")
	return 0, errors.New("unreachable")
}

// stubPrompt makes the terminal interactive (or not) and feeds answer.
func stubPrompt(t *testing.T, interactive bool, answer string) {
	t.Helper()
	oldIn, oldFn, oldJSON, oldQuiet := promptReader, promptInteractiveFn, outputJSON, outputQuiet
	t.Cleanup(func() { promptReader, promptInteractiveFn, outputJSON, outputQuiet = oldIn, oldFn, oldJSON, oldQuiet })
	promptInteractiveFn = func() bool { return interactive }
	if interactive {
		promptReader = strings.NewReader(answer)
	} else {
		promptReader = failReader{t}
	}
}

func TestAskYesNo_DefaultsAndAnswers(t *testing.T) {
	cases := []struct {
		in         string
		defaultYes bool
		want       bool
	}{
		{"\n", true, true}, {"\n", false, false}, {"n\n", true, false},
		{"sí\n", false, true}, {"Y\n", false, true}, {"", true, false}, // EOF answers no
	}
	for _, c := range cases {
		stubPrompt(t, true, c.in)
		var got bool
		_, _ = captureStdout(t, func() error { got = askYesNo("?", c.defaultYes); return nil })
		if got != c.want {
			t.Errorf("askYesNo(%q, default %v) = %v, want %v", c.in, c.defaultYes, got, c.want)
		}
	}
}

func TestConfirmAction_NeverAsksWithoutTerminalOrWithJSONOrQuiet(t *testing.T) {
	stubPrompt(t, false, "")
	if confirmAction("?", false, true) {
		t.Error("non-interactive: want no")
	}
	if !confirmAction("?", true, true) {
		t.Error("--yes: want yes without asking")
	}
	stubPrompt(t, true, "")
	promptReader = failReader{t}
	outputJSON = true
	if confirmAction("?", false, true) {
		t.Error("--json: want no without asking")
	}
	outputJSON, outputQuiet = false, true
	if confirmAction("?", false, true) {
		t.Error("-q: want no without asking")
	}
}

func TestOfferHelperInstall_NonInteractivePrintsGuidance(t *testing.T) {
	installs := stubHelper(t, helperMissing, nil)
	stubPrompt(t, false, "")
	var ok bool
	out, _ := captureStdout(t, func() error { ok = offerHelperInstall(helperMissing, false); return nil })
	if ok || *installs != 0 {
		t.Fatalf("ok=%v installs=%d, want no install without a terminal", ok, *installs)
	}
	if !strings.Contains(out, ct("service.error.not_installed")) {
		t.Errorf("no guidance line:\n%s", out)
	}
}

func TestOfferHelperInstall_DefaultYesInstalls(t *testing.T) {
	installs := stubHelper(t, helperOutdated, nil)
	stubPrompt(t, true, "\n")
	var ok bool
	_, _ = captureStdout(t, func() error { ok = offerHelperInstall(helperOutdated, false); return nil })
	if !ok || *installs != 1 {
		t.Fatalf("ok=%v installs=%d, want one install on Enter", ok, *installs)
	}
}

func TestOfferHelperInstall_ReadyDoesNothing(t *testing.T) {
	installs := stubHelper(t, helperReady, nil)
	stubPrompt(t, false, "")
	if !offerHelperInstall(helperReady, false) || *installs != 0 {
		t.Fatal("a ready helper needs no install")
	}
}

// stubFinishInit records what finishInit runs.
func stubFinishInit(t *testing.T, daemonUp bool) (ups *[][]string, restarts *int) {
	t.Helper()
	stubUp(t)
	oldLoad, oldRun, oldRestart, oldYes, oldPath := loadConfigFn, runUpFn, restartDaemonFn, initYes, cfgPath
	t.Cleanup(func() {
		loadConfigFn, runUpFn, restartDaemonFn, initYes, cfgPath = oldLoad, oldRun, oldRestart, oldYes, oldPath
	})
	ups, restarts = &[][]string{}, new(int)
	loadConfigFn = func() error { return nil }
	runUpFn = func(_ context.Context, targets []string) error { *ups = append(*ups, targets); return nil }
	restartDaemonFn = func(string) error { *restarts++; return nil }
	daemonProbeFn = func(string) bool { return daemonUp }
	return ups, restarts
}

func initWritten() *config.Config {
	return &config.Config{VPNs: map[string]config.VPNConfig{"office": {Type: "wireguard"}}}
}

func TestFinishInit_NonInteractiveDoesNotConnect(t *testing.T) {
	ups, _ := stubFinishInit(t, false)
	stubHelper(t, helperReady, nil)
	stubPrompt(t, false, "")
	if err := finishInit("x.yaml", []profileSummary{{name: "office"}}, initWritten()); err != nil {
		t.Fatal(err)
	}
	if len(*ups) != 0 {
		t.Fatalf("connected without a terminal: %v", *ups)
	}
}

func TestFinishInit_YesInstallsHelperAndConnects(t *testing.T) {
	ups, restarts := stubFinishInit(t, false)
	installs := stubHelper(t, helperMissing, nil)
	stubPrompt(t, false, "")
	initYes = true
	_, err := captureStdout(t, func() error {
		return finishInit("x.yaml", []profileSummary{{name: "office"}, {name: "gone"}}, initWritten())
	})
	if err != nil {
		t.Fatal(err)
	}
	if *installs != 1 || *restarts != 0 {
		t.Fatalf("installs=%d restarts=%d", *installs, *restarts)
	}
	if len(*ups) != 1 || strings.Join((*ups)[0], ",") != "office" {
		t.Fatalf("up targets = %v, want [office]", *ups)
	}
}

func TestFinishInit_AnswerYesRestartsRunningDaemonFirst(t *testing.T) {
	ups, restarts := stubFinishInit(t, true)
	stubHelper(t, helperReady, nil)
	stubPrompt(t, true, "s\n")
	_, err := captureStdout(t, func() error {
		return finishInit("x.yaml", []profileSummary{{name: "office"}}, initWritten())
	})
	if err != nil {
		t.Fatal(err)
	}
	if *restarts != 1 || len(*ups) != 1 {
		t.Fatalf("restarts=%d ups=%v, want restart then up", *restarts, *ups)
	}
}

func TestFinishInit_InvalidConfigStops(t *testing.T) {
	ups, _ := stubFinishInit(t, false)
	installs := stubHelper(t, helperMissing, nil)
	initYes = true
	if err := finishInit("x.yaml", []profileSummary{{name: "office"}}, nil); err != nil {
		t.Fatal(err)
	}
	if len(*ups) != 0 || *installs != 0 {
		t.Fatal("an invalid config must not install or connect")
	}
}

// up on an interactive terminal: the helper is missing → offer, install,
// retry the same profile once.
func TestUp_HelperMissingOffersInstallAndRetries(t *testing.T) {
	stubUp(t)
	installs := stubHelper(t, helperMissing, nil)
	stubPrompt(t, true, "\n")
	daemonProbeFn = func(string) bool { return true }
	daemonConnectFn = func(string, string) (bool, error) { return false, nil }
	daemonTunnelStatesFn = func(string) (map[string]monitor.TunnelMetrics, error) {
		if *installs == 0 {
			return map[string]monitor.TunnelMetrics{"office": {
				Status: vpn.StatusError, LastError: "helper not installed", LastErrorKey: "service.error.not_installed",
			}}, nil
		}
		return map[string]monitor.TunnelMetrics{"office": {Status: vpn.StatusConnected}}, nil
	}
	out, err := captureStdout(t, func() error { return runUp(context.Background(), []string{"office"}) })
	if err != nil {
		t.Fatalf("up: %v\n%s", err, out)
	}
	if *installs != 1 {
		t.Fatalf("installs = %d, want 1", *installs)
	}
}

func TestUp_HelperMissingNonInteractiveFailsWithGuidance(t *testing.T) {
	stubUp(t)
	installs := stubHelper(t, helperMissing, nil)
	stubPrompt(t, false, "")
	daemonProbeFn = func(string) bool { return true }
	daemonConnectFn = func(string, string) (bool, error) { return false, nil }
	daemonTunnelStatesFn = func(string) (map[string]monitor.TunnelMetrics, error) {
		return map[string]monitor.TunnelMetrics{"office": {
			Status: vpn.StatusError, LastError: "helper not installed", LastErrorKey: "service.error.not_installed",
		}}, nil
	}
	out, err := captureStdout(t, func() error { return runUp(context.Background(), []string{"office"}) })
	if err == nil || *installs != 0 {
		t.Fatalf("err=%v installs=%d, want a failure and no install", err, *installs)
	}
	if !strings.Contains(out, ct("service.error.not_installed")) {
		t.Errorf("no guidance in output:\n%s", out)
	}
}
