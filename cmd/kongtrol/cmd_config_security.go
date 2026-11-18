package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// config killswitch / config dns-guard: the dashboard's Security toggles in
// the CLI (CLI-GUI parity), mirroring config public-ip. With a running
// daemon they go through its API (POST /api/v1/security/killswitch |
// dnsguard), so the change applies at once; otherwise kongtrol.yaml is
// edited in place keeping comments, and it applies on the next start.

// toggleArg is the on | off | status argument.
type toggleArg int

const (
	toggleInvalid toggleArg = iota
	toggleOn
	toggleOff
	toggleStatus
)

func parseToggleArg(s string) toggleArg {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "true", "si", "sí", "yes":
		return toggleOn
	case "off", "false", "no":
		return toggleOff
	case "status", "estado":
		return toggleStatus
	}
	return toggleInvalid
}

// securityToggle describes one of the two switches.
type securityToggle struct {
	yamlKey  string // under security:
	keyOn    string // i18n line after turning on
	keyOff   string // i18n line after turning off
	keyLabel string // i18n "Kill switch: %s"
	set      func(*daemonctl.Client, bool) error
	fromFile func() bool // configured value when no daemon runs
	// state renders the daemon's runtime state ("" when none).
	state func(daemonctl.SecurityState) (enabled bool, detail string)
}

var killSwitchToggle = securityToggle{
	yamlKey: "kill_switch", keyOn: "cli.killswitch.on", keyOff: "cli.killswitch.off", keyLabel: "cli.killswitch.label",
	set:      (*daemonctl.Client).SetKillSwitch,
	fromFile: func() bool { return cfg != nil && cfg.Security.KillSwitch.Enabled },
	state: func(st daemonctl.SecurityState) (bool, string) {
		if st.KillSwitchStatus == nil {
			return st.KillSwitchEnabled, ""
		}
		key, args := st.KillSwitchStatus.StateKey()
		return st.KillSwitchEnabled, cf(key, args...)
	},
}

var dnsGuardToggle = securityToggle{
	yamlKey: "dns_guard", keyOn: "cli.dnsguard.on", keyOff: "cli.dnsguard.off", keyLabel: "cli.dnsguard.label",
	set:      (*daemonctl.Client).SetDNSGuard,
	fromFile: func() bool { return cfg != nil && cfg.Security.DNSGuard.Enabled },
	state: func(st daemonctl.SecurityState) (bool, string) {
		switch st.DNSGuardState {
		case "active":
			return st.DNSGuardEnabled, ct("cli.dnsguard.state.active")
		case "idle":
			return st.DNSGuardEnabled, ct("cli.dnsguard.state.idle")
		}
		return st.DNSGuardEnabled, ""
	},
}

func (tg securityToggle) command(use string) *cobra.Command {
	return &cobra.Command{
		Use:  use + " <on|off|status>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := parseToggleArg(args[0])
			if arg == toggleInvalid {
				return fmt.Errorf("%s", cf("cli.config.toggle.invalid", args[0]))
			}
			base := daemonAPIBase()
			running := daemonProbeFn(base)
			if arg == toggleStatus {
				return tg.printStatus(base, running)
			}
			enabled := arg == toggleOn
			if running {
				if err := tg.set(daemonClient(base), enabled); err != nil {
					return err
				}
			} else if err := setSecurityToggleInFile(activeCfgPath, tg.yamlKey, enabled); err != nil {
				return err
			}
			if outputJSON {
				return emitJSON(map[string]any{"enabled": enabled, "applied_now": running})
			}
			if enabled {
				fmt.Println(tuiOK(ct(tg.keyOn)))
			} else {
				fmt.Println(tuiOK(ct(tg.keyOff)))
			}
			if !running {
				fmt.Println(tuiInfo(paint(cDim, ct("cli.config.toggle.saved_file"))))
			}
			return nil
		},
	}
}

func (tg securityToggle) printStatus(base string, running bool) error {
	enabled, detail := tg.fromFile(), ""
	if running {
		st, err := daemonClient(base).Security()
		if err != nil {
			return err
		}
		enabled, detail = tg.state(st)
	}
	if outputJSON {
		return emitJSON(map[string]any{"enabled": enabled, "state": detail})
	}
	word := ct("cli.config.toggle.off")
	if enabled {
		word = ct("cli.config.toggle.on")
	}
	fmt.Println(tuiInfo(cf(tg.keyLabel, word)))
	if detail != "" && enabled {
		fmt.Println(tuiInfo(paint(cDim, detail)))
	}
	return nil
}

// setSecurityToggleInFile sets security.<key>.enabled in kongtrol.yaml.
func setSecurityToggleInFile(path, key string, enabled bool) error {
	return editConfigYAML(path, func(root *yaml.Node) {
		setMapping(childMapping(childMapping(root, "security"), key), "enabled", boolNode(enabled))
	})
}

var (
	configKillSwitchCmd = killSwitchToggle.command("killswitch")
	configDNSGuardCmd   = dnsGuardToggle.command("dns-guard")
)

func init() {
	configCmd.AddCommand(configKillSwitchCmd, configDNSGuardCmd)
}
