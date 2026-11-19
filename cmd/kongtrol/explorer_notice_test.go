package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestExplorerNotice(t *testing.T) {
	if cobra.MousetrapHelpText != "" {
		t.Fatalf("cobra's English mousetrap must be off, got %q", cobra.MousetrapHelpText)
	}
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		var out bytes.Buffer
		showExplorerNotice(&out, strings.NewReader("\n"), lang)
		got := out.String()
		if !strings.Contains(got, "Kongtrol-Setup-windows-amd64.exe") || !strings.Contains(got, "https://kongtrol.app") {
			t.Errorf("%s: notice must name the installer and the site, got %q", lang, got)
		}
		if got != i18n.T(lang, "cli.explorer_notice")+"\n" {
			t.Errorf("%s: unexpected output %q", lang, got)
		}
	}
	// EOF instead of Enter must not hang.
	showExplorerNotice(&bytes.Buffer{}, strings.NewReader(""), i18n.EN)
}

func TestStartedFromExplorerNeedsNoArgs(t *testing.T) {
	if startedFromExplorer([]string{"kongtrol", "dashboard"}) {
		t.Fatal("a command with arguments must always run")
	}
}
