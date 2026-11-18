package main

import (
	"bytes"
	"strings"
	"testing"
)

// stubHelper replaces the helper check and installer for one test.
func stubHelper(t *testing.T, need helperNeed, installErr error) *int {
	t.Helper()
	oldState, oldInstall := helperStateFn, installHelperFn
	t.Cleanup(func() { helperStateFn, installHelperFn = oldState, oldInstall })
	installs := 0
	helperStateFn = func() helperNeed { return need }
	installHelperFn = func() error { installs++; return installErr }
	return &installs
}

// kongtrol dashboard is a client of the shared daemon: it reuses (or
// starts) `kongtrol daemon` and only opens the browser on its address.
func TestDashboard_UsesSharedDaemon(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return true }
	spawnDaemonFn = func(string) (<-chan error, error) { t.Fatal("must not spawn when a daemon answers"); return nil, nil }
	oldOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = oldOpen })
	var opened string
	openBrowserFn = func(u string) error { opened = u; return nil }

	out, err := captureStdout(t, func() error { return dashboardCmd.RunE(dashboardCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if opened != daemonAPIBase() {
		t.Fatalf("opened %q, want the daemon's %q", opened, daemonAPIBase())
	}
	if !strings.Contains(out, ct("cli.dashboard.keeps_running")) {
		t.Errorf("output lacks the keeps-running line:\n%s", out)
	}
}

func TestRootVersionFlag_MatchesVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetArgs(nil) })
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), versionText(); got != want {
		t.Fatalf("--version printed %q, want %q (same as kongtrol version)", got, want)
	}
}

func TestAlertBell_LocalOnUpAndStatus_GlobalHidden(t *testing.T) {
	for _, c := range []string{"up", "status"} {
		cmd, _, err := rootCmd.Find([]string{c})
		if err != nil {
			t.Fatal(err)
		}
		if f := cmd.LocalNonPersistentFlags().Lookup("alert-bell"); f == nil || f.Hidden {
			t.Errorf("%s: want a visible local --alert-bell", c)
		}
	}
	if f := rootCmd.PersistentFlags().Lookup("alert-bell"); f == nil || !f.Hidden {
		t.Error("global --alert-bell must stay as a hidden alias")
	}
}

func TestDeprecatedGlobalAlertBell_Warns(t *testing.T) {
	stubUp(t)
	pf := rootCmd.PersistentFlags()
	t.Cleanup(func() { _ = pf.Set("alert-bell", "false"); pf.Lookup("alert-bell").Changed = false; alertBell = false })
	if err := pf.Set("alert-bell", "true"); err != nil {
		t.Fatal(err)
	}
	stderr := captureStderr(t, func() { _ = rootCmd.PersistentPreRunE(versionCmd, nil) })
	if !strings.Contains(stderr, "--alert-bell") || !strings.Contains(stderr, "kongtrol up --alert-bell") {
		t.Fatalf("no deprecation line on stderr: %q", stderr)
	}
}

func TestStatusDashboardFlag_DeprecatedAndHidden(t *testing.T) {
	f := statusCmd.Flags().Lookup("dashboard")
	if f == nil || !f.Hidden {
		t.Fatal("status --dashboard must stay as a hidden flag")
	}
}

func TestDoctorFix_InstallsMissingHelper(t *testing.T) {
	stubUp(t)
	installs := stubHelper(t, helperMissing, nil)
	if _, err := captureStdout(t, func() error { _, err := applyDoctorFixes(); return err }); err != nil {
		t.Fatal(err)
	}
	if *installs != 1 {
		t.Fatalf("installs = %d, want 1", *installs)
	}

	installs = stubHelper(t, helperReady, nil)
	_, _ = captureStdout(t, func() error { _, err := applyDoctorFixes(); return err })
	if *installs != 0 {
		t.Fatalf("installed a helper that already answers (%d)", *installs)
	}
}
