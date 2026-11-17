package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// englishWords flags English left in Spanish help (command names and
// examples are excluded: only prose is checked).
var englishWords = regexp.MustCompile(`(?i)\b(the|and|with|your|for|to|of|this|shows?|lists?|runs?|help|usage|flags?|available|default)\b`)

// sharedText matches the parts every language shares (kongtrol commands,
// flags, placeholders, file names) so only translated prose remains.
var sharedText = regexp.MustCompile("kongtrol(\\s+[a-z][a-z-]*)*|--?[a-z-]+|<[^>]*>|\\[[^]]*\\]|[a-z0-9_.]+\\.(yaml|txt)|KONGTROL_LANG")

type helpText struct{ use, short, long, example string }

func collectHelp(t *testing.T, lang i18n.Lang) map[string]helpText {
	t.Helper()
	setupHelp(rootCmd, lang)
	out := map[string]helpText{}
	walkVisible(rootCmd, func(c *cobra.Command) {
		if c != rootCmd && !c.IsAvailableCommand() && c.Name() != "help" {
			return
		}
		out[c.CommandPath()] = helpText{c.Use, c.Short, c.Long, c.Example}
	})
	return out
}

func TestHelpCoversEveryCommand(t *testing.T) {
	defer setupHelp(rootCmd, cliLang())
	es := collectHelp(t, i18n.ES)
	en := collectHelp(t, i18n.EN)
	if len(es) < 30 || len(es) != len(en) {
		t.Fatalf("walked %d ES / %d EN commands", len(es), len(en))
	}
	for path, e := range es {
		n := en[path]
		for field, pair := range map[string][2]string{"short": {e.short, n.short}, "long": {e.long, n.long}, "example": {e.example, n.example}} {
			if strings.TrimSpace(pair[0]) == "" || strings.TrimSpace(pair[1]) == "" || strings.HasPrefix(pair[0], "help.") || strings.HasPrefix(pair[1], "help.") {
				t.Errorf("%s: %s missing (es=%q en=%q)", path, field, pair[0], pair[1])
			}
		}
		if e.short == n.short || e.long == n.long {
			t.Errorf("%s: Short/Long not translated (es=%q / %q)", path, e.short, e.long)
		}
		if strings.Contains(e.long, "\n") || strings.Contains(n.long, "\n") {
			t.Errorf("%s: Long must be one line", path)
		}
		if lines := strings.Count(strings.TrimSpace(e.example), "\n") + 1; lines > 3 {
			t.Errorf("%s: %d example lines, want 1-3", path, lines)
		}
		for _, text := range []string{e.short, e.long, strings.TrimPrefix(e.use, path[strings.LastIndex(path, " ")+1:])} {
			if w := englishWords.FindString(sharedText.ReplaceAllString(text, " ")); w != "" {
				t.Errorf("%s: English word %q in Spanish help: %q", path, w, text)
			}
		}
	}
}

func TestHelpOutputTranslatedAndGrouped(t *testing.T) {
	defer setupHelp(rootCmd, cliLang())
	for _, tc := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.ES, []string{"Uso:", "Conexión:", "Configuración:", "Diagnóstico:", "Sistema:", "Opciones:", "muestra la ayuda de kongtrol", "Guía completa: https://kongtrol.app/guia"}},
		{i18n.EN, []string{"Usage:", "Connection:", "Setup:", "Diagnostics:", "System:", "Flags:", "help for kongtrol", "Full guide: https://kongtrol.app/en/guide"}},
	} {
		setupHelp(rootCmd, tc.lang)
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		if err := rootCmd.Help(); err != nil {
			t.Fatal(err)
		}
		rootCmd.SetOut(nil)
		out := buf.String()
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s help lacks %q:\n%s", tc.lang, w, out)
			}
		}
		if tc.lang == i18n.ES {
			for _, w := range []string{"Available Commands", "Additional", "Flags:", "help for", "Help about"} {
				if strings.Contains(out, w) {
					t.Errorf("ES help still has %q", w)
				}
			}
		}
		for _, hidden := range []string{"daemon", "helper"} {
			if regexp.MustCompile(`(?m)^\s+` + hidden + `\s`).MatchString(out) {
				t.Errorf("%s help lists hidden command %q", tc.lang, hidden)
			}
		}
	}
	// Every visible top-level command sits in a group, in a fixed order.
	if !rootCmd.AllChildCommandsHaveGroup() {
		t.Error("some top-level command has no help group")
	}
}
