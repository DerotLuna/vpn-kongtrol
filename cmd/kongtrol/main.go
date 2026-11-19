// Command kongtrol is the Kongtrol CLI.
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
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/nordvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/surfshark"
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
	col                 *monitor.Collector
	watchdog            *monitor.Watchdog
	scheduler           *monitor.Scheduler
	dnsMgr              *monitor.DNSManager
	policyResolver      *monitor.PolicyResolver
	splitDNSMgr         *monitor.SplitDNSManager
	destBlocker         security.DestinationBlocker
	daemonEvents        = monitor.NewEventLog(100)
	policyWatch         *monitor.PolicyWatch
	alertBell           bool
	apiToken            string
	sessionGreetingLine string
	sessionLastUseLine  string
	// allowSetupMode lets loadConfig start with no config file (daemon
	// only); setupMode is true while the daemon runs that way.
	allowSetupMode bool
	setupMode      bool
)

var rootCmd = &cobra.Command{
	Use: "kongtrol",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputPlain {
			_ = os.Setenv("NO_COLOR", "1")
		}
		// The global --alert-bell is a deprecated alias: up and status own
		// a local --alert-bell that shadows it (daemon gets a hidden one).
		if cmd.Root().PersistentFlags().Changed("alert-bell") {
			warnDeprecated("--alert-bell", "kongtrol up --alert-bell")
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
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "doctor" || prefsOnlyCommand(cmd) {
			return nil
		}
		// Only the daemon may start with no kongtrol.yaml (first-run setup
		// from the dashboard); every other command still asks for
		// `kongtrol init`.
		allowSetupMode = cmd.Name() == "daemon" && cmd.Parent() != nil && !cmd.Parent().HasParent()
		return loadConfig()
	},
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SuggestionsMinimumDistance = 2
	rootCmd.SetFlagErrorFunc(flagError)

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", ct("cli.flag.config"))
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, ct("cli.flag.json"))
	rootCmd.PersistentFlags().BoolVarP(&outputQuiet, "quiet", "q", false, ct("cli.flag.quiet"))
	rootCmd.PersistentFlags().BoolVar(&outputPlain, "plain", false, ct("cli.flag.plain"))
	rootCmd.PersistentFlags().BoolVar(&alertBell, "alert-bell", false, "")
	_ = rootCmd.PersistentFlags().MarkHidden("alert-bell")

	// --version prints what `kongtrol version` prints. The flag is declared
	// here so its usage text is an i18n string (cobra only adds its own
	// English one when the flag is missing); cobra handles it before
	// PersistentPreRunE, so the template renders the whole output.
	rootCmd.Version = version
	rootCmd.Flags().BoolP("version", "v", false, ct("cli.flag.version"))
	cobra.AddTemplateFunc("kongtrolVersion", versionText)
	rootCmd.SetVersionTemplate("{{kongtrolVersion}}")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(routesCmd)
	rootCmd.AddCommand(ipCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(mapCmd)
	rootCmd.AddCommand(dashboardCmd)
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
	setupHelp(rootCmd, cliLang())
	if cmd, err := rootCmd.ExecuteC(); err != nil {
		if !isReported(err) {
			fmt.Fprintln(os.Stderr, tuiErr(userFacingError(cmd, err)))
		}
		os.Exit(1)
	}
}
