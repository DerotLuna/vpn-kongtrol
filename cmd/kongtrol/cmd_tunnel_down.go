package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
)

var profileTunnelDownCmd = &cobra.Command{
	Use:  "on-tunnel-down <name> <allow|block|connect>",
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		name, action := strings.TrimSpace(args[0]), strings.ToLower(strings.TrimSpace(args[1]))
		if action == "" {
			return localizeConfigErr(&config.ValidationError{Key: "config.err.profile_on_tunnel_down", Args: []any{name, action}})
		}
		if base := daemonAPIBase(); daemonProbeFn(base) {
			if err := configClient(base).UpdateVPN(name, configedit.VPNProfile{OnTunnelDown: &action}); err != nil {
				return remoteUserErr(err)
			}
		} else if err := editConfigOffline(func(c *config.Config) error {
			profile, ok := c.VPNs[name]
			if !ok {
				return &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}}
			}
			profile.OnTunnelDown = action
			c.VPNs[name] = profile
			return nil
		}); err != nil {
			return err
		}
		return printTunnelDownChange(name, action)
	},
}

var policyTunnelDownCmd = &cobra.Command{
	Use:  "on-tunnel-down <name> <inherit|allow|block|connect>",
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		name, action := strings.TrimSpace(args[0]), strings.ToLower(strings.TrimSpace(args[1]))
		override := action
		if action == "inherit" {
			override = ""
		} else if action == "" {
			return localizeConfigErr(&config.ValidationError{Key: "config.err.on_tunnel_down", Args: []any{name, action}})
		}
		if base := daemonAPIBase(); daemonProbeFn(base) {
			if err := configClient(base).UpdatePolicy(name, configedit.PolicyPatch{OnTunnelDown: &override}); err != nil {
				return remoteUserErr(err)
			}
		} else if err := editConfigOffline(func(c *config.Config) error {
			i := configedit.FindPolicy(c, name)
			if i < 0 {
				return &config.ValidationError{Key: configedit.ErrKeyPolicyNotFound, Args: []any{name}}
			}
			c.Policies[i].OnTunnelDown = override
			return nil
		}); err != nil {
			return err
		}
		return printTunnelDownChange(name, action)
	},
}

func printTunnelDownChange(name, action string) error {
	if outputJSON {
		return emitJSON(map[string]string{"name": name, "on_tunnel_down": action})
	}
	if !outputQuiet {
		fmt.Println(tuiOK(cf("cli.on_tunnel_down.saved", name, action)))
	}
	return nil
}

func init() {
	profileCmd.AddCommand(profileTunnelDownCmd)
	policyCmd.AddCommand(policyTunnelDownCmd)
}
