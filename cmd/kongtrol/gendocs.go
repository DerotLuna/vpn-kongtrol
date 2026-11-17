package main

//go:generate go run . gen-docs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/docgen"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// genDocsCmd regenerates the reference docs from the code (hidden, for
// `go generate ./...` / `make docs`). --check only verifies, for CI.
var genDocsCmd = &cobra.Command{
	Use:               "gen-docs",
	Hidden:            true,
	Args:              cobra.NoArgs,
	PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	RunE: func(c *cobra.Command, _ []string) error {
		root, err := repoRoot()
		if err != nil {
			return err
		}
		check, _ := c.Flags().GetBool("check")
		files, err := generatedDocs()
		if err != nil {
			return err
		}
		if check {
			// Report only, never fail: docs don't block merges (AGENTS.md).
			// In GitHub Actions each problem becomes a ::warning annotation.
			var problems []string
			for _, f := range files {
				if err := docgen.Check(root, f); err != nil {
					problems = append(problems, err.Error())
				}
			}
			problems = append(problems, docgen.MissingDescriptions()...)
			for _, p := range problems {
				if os.Getenv("GITHUB_ACTIONS") == "true" {
					fmt.Printf("::warning title=Docs out of date::%s\n", p)
				} else {
					fmt.Println("warning:", p)
				}
			}
			if len(problems) == 0 {
				fmt.Println("generated docs are up to date")
			}
			return nil
		}
		today := time.Now().Format("2006-01-02")
		for _, f := range files {
			changed, err := docgen.Update(root, f, today)
			if err != nil {
				return err
			}
			if changed {
				fmt.Println("wrote", f.Path)
			}
		}
		return nil
	},
}

func init() {
	genDocsCmd.Flags().Bool("check", false, "")
	rootCmd.AddCommand(genDocsCmd)
}

// generatedDocs builds every generated reference file from the code.
func generatedDocs() ([]docgen.File, error) {
	defer setupHelp(rootCmd, cliLang())
	var refs []docgen.CLIRef
	for _, lang := range i18n.Langs() {
		setupHelp(rootCmd, lang)
		refs = append(refs, docgen.BuildCLI(rootCmd, lang))
	}
	files := []docgen.File{{
		Path:   "docs/cli.json",
		Render: func(s docgen.Stamp) ([]byte, error) { return docgen.CLIJSON(refs, s) },
	}}
	for _, ref := range refs {
		suffix := ".md"
		if ref.Lang != i18n.ES {
			suffix = "." + string(ref.Lang) + ".md"
		}
		files = append(files,
			docgen.File{Path: "docs/CLI" + suffix, Render: func(s docgen.Stamp) ([]byte, error) { return docgen.CLIMarkdown(ref, s), nil }},
			docgen.File{Path: "docs/CONFIG" + suffix, Render: func(s docgen.Stamp) ([]byte, error) { return docgen.ConfigMarkdown(ref.Lang, s) }},
		)
	}
	return files, nil
}

// repoRoot walks up from the working directory to the go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("gen-docs: go.mod not found above the working directory")
		}
		dir = parent
	}
}
