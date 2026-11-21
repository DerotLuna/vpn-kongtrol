package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
)

var routesCheckDefaultCmd = &cobra.Command{
	Use:  "check-default <profile>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDefaultEgress(cmd, args[0], nil)
	},
}

var routesDefaultEgressCmd = &cobra.Command{
	Use:  "default-egress <profile> <on|off>",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled := args[1] == "on"
		if !enabled && args[1] != "off" {
			return fmt.Errorf("%s", ct("cli.routes.default.invalid_choice"))
		}
		return runDefaultEgress(cmd, args[0], &enabled)
	},
}

func runDefaultEgress(cmd *cobra.Command, name string, enabled *bool) error {
	result, err := profileDefaultEgress(name, enabled)
	if err != nil {
		return err
	}
	if outputJSON {
		return emitJSON(result)
	}
	if outputQuiet {
		return nil
	}
	switch {
	case enabled != nil && !*enabled:
		fmt.Fprintln(cmd.OutOrStdout(), cf("cli.routes.default.disabled", name))
	case !result.Compatible:
		err := &config.ValidationError{Key: "config.err.default_egress_" + result.Reason}
		fmt.Fprintln(cmd.OutOrStdout(), localizeConfigErr(err))
	case result.Enabled:
		fmt.Fprintln(cmd.OutOrStdout(), cf("cli.routes.default.enabled", name))
	default:
		fmt.Fprintln(cmd.OutOrStdout(), cf("cli.routes.default.choose", name))
	}
	return nil
}

func profileDefaultEgress(name string, enabled *bool) (config.DefaultEgressStatus, error) {
	if base := daemonAPIBase(); cfgPath == "" && daemonProbeFn(base) {
		result, err := configClient(base).DefaultEgress(name, enabled)
		return result, remoteUserErr(err)
	}
	var result config.DefaultEgressStatus
	current, err := config.Load(activeCfgPath)
	if err != nil {
		return result, localizeConfigErr(err)
	}
	if _, ok := current.VPNs[name]; !ok {
		return result, localizeConfigErr(&config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
	}
	if enabled != nil {
		err := configedit.SetDefaultEgress(current, name, *enabled, func(next *config.Config) error {
			_, err := configedit.Save(activeCfgPath, next)
			return err
		})
		if err != nil {
			return result, localizeConfigErr(err)
		}
	}
	return current.VPNs[name].DefaultEgressStatus(), nil
}

func init() {
	routesCmd.AddCommand(routesCheckDefaultCmd, routesDefaultEgressCmd)
}
