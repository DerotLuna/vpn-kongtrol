package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ── audit ─────────────────────────────────────────────────────────────────────

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Manage the audit log",
}

// ── config ────────────────────────────────────────────────────────────────────

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config management",
}

var configFavCmd = &cobra.Command{
	Use:     "favorites",
	Aliases: []string{"fav"},
	Short:   "Manage favorite VPN profiles",
}

var configFavListCmd = &cobra.Command{
	Use:   "list",
	Short: "List favorite profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Favorites []string `json:"favorites"`
			}{Favorites: p.Favorites})
		}
		if len(p.Favorites) == 0 {
			fmt.Println(styleDim.Render(ct("cli.favorites.none")))
			return nil
		}
		for _, name := range p.Favorites {
			fmt.Println("  " + styleGold.Render(sym("★", "*")) + "  " + styleBright.Render(name))
		}
		return nil
	},
}

var configFavAddCmd = &cobra.Command{
	Use:   "add <profile>",
	Short: "Add favorite profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := addFavorite(args[0]); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.favorites.added", args[0])))
		return nil
	},
}

var configFavRemoveCmd = &cobra.Command{
	Use:   "remove <profile>",
	Short: "Remove favorite profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := removeFavorite(args[0]); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.favorites.removed", args[0])))
		return nil
	},
}

var configDefaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Manage default CLI behavior",
}

var configDefaultsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				DefaultGroup string `json:"default_group"`
			}{DefaultGroup: p.DefaultGroup})
		}
		if p.DefaultGroup == "" {
			fmt.Println(styleDim.Render(ct("cli.defaults.none")))
			return nil
		}
		fmt.Println(tuiInfo(cf("cli.defaults.group", p.DefaultGroup)))
		return nil
	},
}

var configDefaultsSetGroupCmd = &cobra.Command{
	Use:   "set-group <group>",
	Short: "Set default group for 'kongtrol up'",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DefaultGroup = args[0]
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.defaults.group_set", args[0])))
		return nil
	},
}

var configLangCmd = &cobra.Command{
	Use:   "lang <es|en>",
	Short: "Set the CLI display language",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lang := strings.ToLower(strings.TrimSpace(args[0]))
		if lang != "es" && lang != "en" {
			return fmt.Errorf("%s", cf("cli.lang.invalid", args[0]))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.Language = lang
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.lang.set", lang)))
		return nil
	},
}

var configDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Manage the dashboard's local bind/port override",
}

var configDashboardShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the dashboard bind/port override",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Port int    `json:"port,omitempty"`
				Bind string `json:"bind,omitempty"`
			}{Port: p.DashboardPort, Bind: p.DashboardBind})
		}
		if p.DashboardPort == 0 && p.DashboardBind == "" {
			fmt.Println(styleDim.Render(ct("cli.config.dashboard.no_override")))
			return nil
		}
		if p.DashboardPort != 0 {
			fmt.Println(tuiInfo(cf("cli.config.dashboard.port", p.DashboardPort)))
		}
		if p.DashboardBind != "" {
			fmt.Println(tuiInfo(cf("cli.config.dashboard.bind", p.DashboardBind)))
		}
		return nil
	},
}

var configDashboardSetPortCmd = &cobra.Command{
	Use:   "set-port <port>",
	Short: "Override the dashboard's local port (applies on next restart)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil || port <= 0 || port > 65535 {
			return fmt.Errorf("%s", cf("cli.config.dashboard.invalid_port", args[0]))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardPort = port
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.config.dashboard.port_set", port)))
		return nil
	},
}

var configDashboardSetBindCmd = &cobra.Command{
	Use:   "set-bind <address>",
	Short: "Override the dashboard's bind address (applies on next restart)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bind := strings.TrimSpace(args[0])
		if err := config.ValidateDashboardBind(bind); err != nil {
			return fmt.Errorf("%s", cf("cli.config.dashboard.invalid_bind", bind))
		}
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardBind = bind
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(cf("cli.config.dashboard.bind_set", bind)))
		return nil
	},
}

var configDashboardResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Clear the dashboard bind/port override",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := loadPreferences()
		if err != nil {
			return err
		}
		p.DashboardPort = 0
		p.DashboardBind = ""
		if err := savePreferences(p); err != nil {
			return err
		}
		fmt.Println(tuiOK(ct("cli.config.dashboard.reset")))
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate kongtrol.yaml without connecting",
	RunE: func(cmd *cobra.Command, args []string) error {
		var spin *spinner
		if !outputJSON {
			spin = newSpinner(ct("cli.config.validate.spinner"))
			spin.Start()
		}

		validatedCfg, err := config.Load(cfgPath)
		if spin != nil {
			spin.Stop()
		}
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(struct {
				Valid    bool `json:"valid"`
				Profiles int  `json:"profiles"`
				Policies int  `json:"policies"`
				Groups   int  `json:"groups"`
			}{
				Valid:    true,
				Profiles: len(validatedCfg.VPNs),
				Policies: len(validatedCfg.Policies),
				Groups:   len(validatedCfg.Groups),
			})
		}
		fmt.Println(tuiOK(styleBright.Render(ct("cli.config.validate.valid"))))
		fmt.Println("  " + styleDim.Render(fmt.Sprintf(
			ct("cli.config.validate.summary"),
			len(validatedCfg.VPNs), len(validatedCfg.Policies), len(validatedCfg.Groups),
		)))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)
	configFavCmd.AddCommand(configFavListCmd, configFavAddCmd, configFavRemoveCmd)
	configDefaultsCmd.AddCommand(configDefaultsShowCmd, configDefaultsSetGroupCmd)
	configDashboardCmd.AddCommand(configDashboardShowCmd, configDashboardSetPortCmd, configDashboardSetBindCmd, configDashboardResetCmd)
	configCmd.AddCommand(configFavCmd, configDefaultsCmd, configLangCmd, configDashboardCmd)
}
