package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/configexport"
)

// ── export ────────────────────────────────────────────────────────────────────

// exportCmd prints the loaded config as YAML that loads back unchanged
// (internal/configexport, shared with GET /api/v1/config/export). Colours
// only on an interactive terminal, so `kongtrol export > file.yaml` is
// always valid YAML.
var exportCmd = &cobra.Command{
	Use: "export",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("%s", ct("cli.error.no_config_loaded"))
		}
		style := configexport.Style{}
		if isTerminal() && !outputPlain && !outputQuiet {
			style = configexport.Style{
				Section: func(s string) string { return styleGold.Render(s) },
				Key:     func(s string) string { return styleMapVia.Render(s) },
				Value:   func(s string) string { return styleStatusIP.Render(s) },
				Comment: func(s string) string { return styleDim.Render(s) },
			}
		}
		return configexport.Write(os.Stdout, cfg, cliLang(), style)
	},
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		// Piped / quiet output stays a bare, parseable version string.
		if outputQuiet || outputPlain || !isTerminal() {
			fmt.Println(version)
			return
		}
		PrintHeader(version)
	},
}
