package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func withProfiles(t *testing.T, names ...string) {
	t.Helper()
	t.Setenv("KONGTROL_LANG", "es")
	old := cfg
	t.Cleanup(func() { cfg = old })
	cfg = &config.Config{VPNs: map[string]config.VPNConfig{}}
	for _, n := range names {
		cfg.VPNs[n] = config.VPNConfig{}
	}
}

func TestValidateProfileNames(t *testing.T) {
	withProfiles(t, "oficina", "casa")

	if err := validateProfileNames([]string{"oficina", "casa"}); err != nil {
		t.Fatalf("known profiles rejected: %v", err)
	}

	err := validateProfileNames([]string{"oficna"})
	if err == nil {
		t.Fatal("expected error for a mistyped profile")
	}
	msg := err.Error()
	for _, want := range []string{"«oficna»", "«oficina»", "casa, oficina"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}

	err = validateProfileNames([]string{"zzzzzz"})
	if err == nil || strings.Contains(err.Error(), "¿") || !strings.Contains(err.Error(), "casa, oficina") {
		t.Fatalf("unexpected message for unrelated name: %v", err)
	}
	if strings.Contains(err.Error(), "%!") {
		t.Fatalf("format mismatch: %q", err.Error())
	}
}

func TestResolveProfiles_ValidatesBeforeWork(t *testing.T) {
	withProfiles(t, "oficina")
	if _, err := resolveProfiles("down", []string{"x"}, ""); err == nil {
		t.Fatal("expected unknown profile error")
	}
	_, err := resolveProfiles("up", nil, "")
	if err == nil || !strings.Contains(err.Error(), "kongtrol up oficina") {
		t.Fatalf("expected a which-VPN example, got %v", err)
	}
}

func TestAlreadyReported(t *testing.T) {
	base := errors.New("boom")
	wrapped := alreadyReported(fmt.Errorf("up x: %w", base))
	if !isReported(wrapped) {
		t.Fatal("wrapped error must be reported")
	}
	if !errors.Is(wrapped, base) {
		t.Fatal("wrapper must keep the error chain")
	}
	if isReported(base) {
		t.Fatal("plain error must not be reported")
	}
	if alreadyReported(nil) != nil {
		t.Fatal("nil stays nil")
	}
}

func TestUserFacingError_TranslatesCobra(t *testing.T) {
	t.Setenv("KONGTROL_LANG", "es")
	cmd := &cobra.Command{Use: "up"}
	got := userFacingError(cmd, errors.New("unknown command \"upp\" for \"kongtrol\"\n\nDid you mean this?\n\tup\n"))
	if !strings.Contains(got, "«upp»") || !strings.Contains(got, "kongtrol up") {
		t.Fatalf("unknown command not translated: %q", got)
	}
	got = userFacingError(cmd, errors.New("accepts 1 arg(s), received 0"))
	if !strings.Contains(got, "up --help") {
		t.Fatalf("args error not translated: %q", got)
	}
	got = flagError(cmd, errors.New("unknown flag: --foo")).Error()
	if !strings.Contains(got, "--foo") || !strings.Contains(got, "up --help") || strings.Contains(got, "unknown flag") {
		t.Fatalf("flag error not translated: %q", got)
	}
}

func TestPrefsOnlyCommandsSkipConfig(t *testing.T) {
	for _, c := range []*cobra.Command{configLangCmd, configFavAddCmd, configDashboardSetPortCmd} {
		if !prefsOnlyCommand(c) {
			t.Fatalf("%s must not require kongtrol.yaml", c.CommandPath())
		}
	}
	if prefsOnlyCommand(configValidateCmd) {
		t.Fatal("config validate needs kongtrol.yaml")
	}
}
