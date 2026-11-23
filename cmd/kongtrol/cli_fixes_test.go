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
	installHelperFn = func(bool) error { installs++; return installErr }
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
	oldPanel := daemonPanelURLFn
	t.Cleanup(func() { daemonPanelURLFn = oldPanel })
	daemonPanelURLFn = func(base, path string) string { return base + path + "#k=nonce" }

	out, err := captureStdout(t, func() error { return dashboardCmd.RunE(dashboardCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	// Opens the daemon's page with a one-time sign-in nonce, never the
	// plain address (the page no longer hands out credentials).
	if want := daemonAPIBase() + "/#k=nonce"; opened != want {
		t.Fatalf("opened %q, want %q", opened, want)
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

// Uninstall refuses sudo: HOME may be root's or the user's depending on the OS.
func TestRunViaSudo(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if !runViaSudo(0, env(map[string]string{"SUDO_USER": "ana"})) {
		t.Error("sudo from ana not detected")
	}
	if runViaSudo(0, env(nil)) || runViaSudo(0, env(map[string]string{"SUDO_USER": "root"})) {
		t.Error("plain root must be allowed")
	}
	if runViaSudo(1000, env(map[string]string{"SUDO_USER": "ana"})) || runViaSudo(-1, env(map[string]string{"SUDO_USER": "ana"})) {
		t.Error("non-root must be allowed")
	}
	// ADR-0100: the other ways to become root for another user.
	for _, m := range []map[string]string{
		{"SUDO_UID": "1000"},
		{"DOAS_USER": "ana"},
		{"PKEXEC_UID": "1000"},
	} {
		if !runViaSudo(0, env(m)) {
			t.Errorf("%v not detected", m)
		}
	}
	if runViaSudo(0, env(map[string]string{"SUDO_UID": "0", "DOAS_USER": "root", "PKEXEC_UID": "0"})) {
		t.Error("root acting for root must be allowed")
	}
}
