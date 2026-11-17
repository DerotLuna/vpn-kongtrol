package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vpn-kongtrol/kongtrol/internal/docgen"
)

// TestGeneratedDocsUpToDate compares docs/CLI*.md, docs/CONFIG*.md and
// docs/cli.json with a fresh generation. Docs never block a merge
// (AGENTS.md): a stale file is logged as a warning, and fails only with
// KONGTROL_DOCS_STRICT=1. Fix: go generate ./... (or make docs).
func TestGeneratedDocsUpToDate(t *testing.T) {
	files, err := generatedDocs()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := docgen.Check(filepath.Join("..", ".."), f); err != nil {
			if os.Getenv("KONGTROL_DOCS_STRICT") == "1" {
				t.Error(err)
			} else {
				t.Log("warning:", err)
			}
		}
	}
}

// TestFlagsTranslatable: every visible flag's usage maps back to an i18n
// key, so --help and the generated docs show it in every language.
func TestFlagsTranslatable(t *testing.T) {
	defer setupHelp(rootCmd, cliLang())
	setupHelp(rootCmd, cliLang())
	walkVisible(rootCmd, func(c *cobra.Command) {
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden || f.Name == "help" {
				return
			}
			if len(f.Annotations[annotationFlagKey]) != 1 {
				t.Errorf("%s --%s: usage %q is not an internal/i18n text", c.CommandPath(), f.Name, f.Usage)
			}
		})
	})
}
