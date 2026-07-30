// Command kongtrol is the VPN Kongtrol CLI.
// It orchestrates multiple VPN connections, controls traffic routing,
// enforces security policies, and exposes a monitoring dashboard.
package main

import (
	"fmt"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"

	// Adapter registrations — order is irrelevant; all run via init().
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/ciscoanyconnect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/cloudflarewarp"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/globalprotect"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/tailscale"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3".
// Falls back to a local dev marker when built without ldflags.
var version = "v1.0.1-dev"

var (
	cfgPath             string
	activeCfgPath       string
	cfg                 *config.Config
	adapters            map[string]vpn.VPNAdapter
	routeMgr            routing.RouteManager
	engine              *policy.Engine
	ks                  security.KillSwitch
	killSwitchSvc       *app.KillSwitchService
	profileSvc          atomic.Pointer[app.ProfileService]
	leak                *security.LeakTester
	audit               *security.AuditLogger
	col                 *monitor.Collector
	watchdog            *monitor.Watchdog
	scheduler           *monitor.Scheduler
	dnsMgr              *monitor.DNSManager
	policyResolver      *monitor.PolicyResolver
	splitDNSMgr         *monitor.SplitDNSManager
	alertBell           bool
	apiToken            string
	sessionGreetingLine string
	sessionLastUseLine  string
)

var rootCmd = &cobra.Command{
	Use:   "kongtrol",
	Short: "Multi-VPN orchestration — route traffic, enforce security, monitor tunnels",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputPlain {
			_ = os.Setenv("NO_COLOR", "1")
		}
		if cmd.Name() != "init" {
			sessionGreetingLine, sessionLastUseLine = prepareSessionBanner(time.Now(), cmd)
		}
		// init shows its own animated logo — skip compact header and session banner there.
		if cmd.Name() != "init" && !outputJSON && !outputQuiet {
			if !(cmd.Name() == "status" && statusWatch) {
				if cmd.Name() != "version" {
					PrintHeader(version)
				}
				printSessionBanner()
			}
		}
		// Skip config load for commands that self-handle config discovery.
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "doctor" {
			return nil
		}
		return loadConfig()
	},
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SuggestionsMinimumDistance = 2
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%s: %w", ct("cli.flag.error"), err)
	})
	rootCmd.Short = ct("cli.root.short")
	rootCmd.Long = ct("cli.root.long")
	rootCmd.Example = ct("cli.root.examples")

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", ct("cli.flag.config"))
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, ct("cli.flag.json"))
	rootCmd.PersistentFlags().BoolVarP(&outputQuiet, "quiet", "q", false, ct("cli.flag.quiet"))
	rootCmd.PersistentFlags().BoolVar(&outputPlain, "plain", false, ct("cli.flag.plain"))
	rootCmd.PersistentFlags().BoolVar(&alertBell, "alert-bell", false, ct("cli.flag.alert_bell"))
	upCmd.Example = ct("cli.up.examples")
	downCmd.Example = ct("cli.down.examples")
	reloadCmd.Example = ct("cli.reload.examples")
	statusCmd.Example = ct("cli.status.examples")
	routesListCmd.Example = ct("cli.routes.list.examples")
	mapCmd.Example = ct("cli.map.examples")
	dashboardCmd.Example = ct("cli.dashboard.examples")
	configValidateCmd.Example = ct("cli.config.validate.examples")
	upCmd.Short = ct("cli.up.short")
	downCmd.Short = ct("cli.down.short")
	reloadCmd.Short = ct("cli.reload.short")
	statusCmd.Short = ct("cli.status.short")
	routesCmd.Short = ct("cli.routes.short")
	routesListCmd.Short = ct("cli.routes.list.short")
	checkCmd.Short = ct("cli.check.short")
	mapCmd.Short = ct("cli.map.short")
	dashboardCmd.Short = ct("cli.dashboard.short")
	auditCmd.Short = ct("cli.audit.short")
	configCmd.Short = ct("cli.config.short")
	configValidateCmd.Short = ct("cli.config.validate.short")
	exportCmd.Short = ct("cli.export.short")
	versionCmd.Short = ct("cli.version.short")
	configFavCmd.Short = ct("cli.favorites.short")
	configFavListCmd.Short = ct("cli.favorites.list.short")
	configFavAddCmd.Short = ct("cli.favorites.add.short")
	configFavRemoveCmd.Short = ct("cli.favorites.remove.short")
	configDefaultsCmd.Short = ct("cli.defaults.short")
	configDefaultsShowCmd.Short = ct("cli.defaults.show.short")
	configDefaultsSetGroupCmd.Short = ct("cli.defaults.set_group.short")
	configLangCmd.Short = ct("cli.lang.short")
	configDashboardCmd.Short = ct("cli.config.dashboard.short")
	configDashboardShowCmd.Short = ct("cli.config.dashboard.show.short")
	configDashboardSetPortCmd.Short = ct("cli.config.dashboard.set_port.short")
	configDashboardSetBindCmd.Short = ct("cli.config.dashboard.set_bind.short")
	configDashboardResetCmd.Short = ct("cli.config.dashboard.reset.short")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(routesCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(mapCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(versionCmd)
}

func sortedAdapterNames(m map[string]vpn.VPNAdapter) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tuiErr(err.Error()))
		os.Exit(1)
	}
}
