package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:               "tray",
		Args:              cobra.NoArgs,
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE:              runTray,
	})
}

func runTray(*cobra.Command, []string) error {
	if privops.IsElevated() {
		return errors.New(ct("cli.tray.elevated"))
	}
	exe, err := daemonctl.FindTray(cliLang())
	if err != nil {
		return guidedError("cli.tray.failed", err, "cli.tray.install_hint")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return guidedError("cli.tray.failed", err, "cli.tray.start_hint")
	}
	say(tuiInfo(ct("cli.tray.starting")))
	if _, err := daemonctl.Spawn(exe, nil, filepath.Join(home, ".kongtrol", "tray.log"), cliLang()); err != nil {
		return guidedError("cli.tray.failed", err, "cli.tray.start_hint")
	}
	if outputJSON {
		return emitJSON(map[string]string{"status": "started"})
	}
	say(tuiOK(ct("cli.tray.started")))
	return nil
}
