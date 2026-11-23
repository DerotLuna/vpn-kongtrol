package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/autostart"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// What Kongtrol does when the computer starts (ADR-0098), the same two
// switches as the panel's Ajustes › "Al iniciar" (PUT /api/v1/startup).
// Neither needs the daemon: autostart is the OS's per-user entry and the
// reconnect choice lives in preferences.json, read at every daemon start.

// parseOnOff reads on/off/status (and their Spanish and yes/no forms).
func parseOnOff(arg string) (on, status, ok bool) {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "on", "true", "si", "sí", "yes":
		return true, false, true
	case "off", "false", "no":
		return false, false, true
	case "status":
		return false, true, true
	}
	return false, false, false
}

// configAutostartCmd: "Abrir Kongtrol al iniciar sesión".
var configAutostartCmd = &cobra.Command{
	Use:  "autostart <on|off|status>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		on, status, ok := parseOnOff(args[0])
		if !ok {
			return fmt.Errorf("%s", cf("cli.startup.invalid", args[0], "autostart"))
		}
		if status {
			st := autostart.Status()
			switch {
			case !st.Supported:
				fmt.Println(tuiInfo(ct("cli.startup.autostart.unsupported")))
			case st.Enabled:
				fmt.Println(tuiInfo(ct("cli.startup.autostart.status_on")))
			default:
				fmt.Println(tuiInfo(ct("cli.startup.autostart.status_off")))
			}
			return nil
		}
		if err := autostart.Set(on); err != nil {
			return localizeConfigErr(api.AutostartError(err))
		}
		if on {
			fmt.Println(tuiOK(ct("cli.startup.autostart.on")))
		} else {
			fmt.Println(tuiOK(ct("cli.startup.autostart.off")))
		}
		return nil
	},
}

// configRestoreCmd: "Al iniciar, volver a conectar las VPN que tenías
// conectadas".
var configRestoreCmd = &cobra.Command{
	Use:  "restore-tunnels <on|off|status>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		on, status, ok := parseOnOff(args[0])
		if !ok {
			return fmt.Errorf("%s", cf("cli.startup.invalid", args[0], "restore-tunnels"))
		}
		if status {
			prefs, err := loadPreferences()
			if err != nil {
				return err
			}
			if prefs.RestoreTunnelsEnabled() {
				fmt.Println(tuiInfo(ct("cli.startup.restore.status_on")))
			} else {
				fmt.Println(tuiInfo(ct("cli.startup.restore.status_off")))
			}
			return nil
		}
		if _, err := config.UpdatePreferences("", func(p *config.Preferences) error {
			p.SetRestoreTunnels(on)
			return nil
		}); err != nil {
			return err
		}
		if on {
			fmt.Println(tuiOK(ct("cli.startup.restore.on")))
		} else {
			fmt.Println(tuiOK(ct("cli.startup.restore.off")))
		}
		return nil
	},
}

func init() {
	for _, c := range []*cobra.Command{configAutostartCmd, configRestoreCmd} {
		c.Annotations = map[string]string{annotationPrefsOnly: "true"}
		configCmd.AddCommand(c)
	}
}
