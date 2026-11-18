package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── reload ───────────────────────────────────────────────────────────────────
//
// kongtrol reload picks up a hand-edited kongtrol.yaml in the running
// Kongtrol (the background daemon) without restarting the whole process. It
// always talks to the running daemon's API (daemonAPIBase / probeDaemonAPI,
// like `status --watch`). There is no local fallback: the point of `reload`
// is to update the *running* daemon's in-memory config, policy engine,
// watchdog and DNS manager, so it fails clearly if no daemon answers.
//
// With no arguments it applies everything: policies, then every connected
// tunnel restarted in place. Profiles the daemon does not know yet (added by
// hand) need a daemon restart (its adapters are built once at boot): reload
// asks and runs `kongtrol restart`'s path; --yes answers for scripts and,
// without a terminal, it prints how to do it.
var (
	reloadGroup      string
	reloadTunnel     string // deprecated --tunnel, now the [perfil] argument
	reloadPolicyOnly bool
	reloadYes        bool
)

var reloadCmd = &cobra.Command{
	Use:  "reload [perfil]",
	Args: cobra.MaximumNArgs(1),
	RunE: runReload,
}

func runReload(cmd *cobra.Command, args []string) error {
	tunnel := ""
	if len(args) == 1 {
		tunnel = args[0]
	}
	if cmd.Flags().Changed("tunnel") {
		warnDeprecated("kongtrol reload --tunnel", "kongtrol reload <perfil>")
		if tunnel == "" {
			tunnel = reloadTunnel
		}
	}
	if tunnel != "" && (reloadGroup != "" || reloadPolicyOnly) {
		return fmt.Errorf("%s", ct("cli.reload.error.one_scope"))
	}
	if tunnel != "" {
		if err := validateProfileNames([]string{tunnel}); err != nil {
			return err
		}
	}
	if reloadGroup != "" && cfg != nil {
		if _, ok := cfg.Groups[reloadGroup]; !ok {
			return fmt.Errorf("%s", cf("cli.error.unknown_group", reloadGroup))
		}
	}
	base := daemonAPIBase()
	if !probeDaemonAPI(base) {
		return fmt.Errorf("%s", ct("cli.reload.error.no_daemon"))
	}

	if err := daemonReloadPolicy(base); err != nil {
		return fmt.Errorf("%s: %w", ct("cli.reload.error.policy"), err)
	}
	fmt.Println(tuiOK(ct("cli.reload.policy_reloaded")))

	switch {
	case reloadPolicyOnly:
		return nil
	case tunnel != "":
		return reloadOneTunnel(base, tunnel)
	case reloadGroup != "":
		return reloadOneGroup(base, reloadGroup)
	}
	return reloadEverything(base)
}

func reloadOneTunnel(base, name string) error {
	result, err := daemonReloadTunnel(base, name)
	if err != nil {
		return fmt.Errorf("%s: %w", cf("cli.reload.error.tunnel", name), err)
	}
	switch {
	case result.RestartRequired:
		return offerRestartForNewProfiles(base, []string{name})
	case len(result.Restarted) == 0:
		fmt.Println(tuiOK(cf("cli.reload.tunnel_skipped", name)))
	default:
		fmt.Println(tuiOK(cf("cli.reload.tunnel_reloaded", name)))
	}
	return nil
}

func reloadOneGroup(base, group string) error {
	result, err := daemonReloadGroup(base, group)
	if err != nil {
		fmt.Println(tuiErr(cf("cli.reload.group_failed", group, err)))
		return alreadyReported(err)
	}
	if result.RestartRequired {
		return offerRestartForNewProfiles(base, result.MissingProfiles)
	}
	fmt.Println(tuiOK(cf("cli.reload.group_reloaded", group)))
	return nil
}

// reloadEverything restarts every connected tunnel in place with the new
// file, then offers a restart when kongtrol.yaml has profiles the daemon
// does not know yet.
func reloadEverything(base string) error {
	statuses, err := daemonTunnelStatusesFn(base)
	if err != nil {
		return err
	}
	var connected []string
	for name, st := range statuses {
		if st == vpn.StatusConnected {
			connected = append(connected, name)
		}
	}
	sort.Strings(connected)
	var missing []string
	for _, name := range connected {
		result, err := daemonReloadTunnel(base, name)
		switch {
		case err != nil:
			fmt.Println(tuiErr(cf("cli.reload.error.tunnel", name) + ": " + err.Error()))
		case result.RestartRequired:
			missing = append(missing, name)
		case len(result.Restarted) > 0:
			fmt.Println(tuiOK(cf("cli.reload.tunnel_reloaded", name)))
		}
	}
	if cfg != nil {
		for name := range cfg.VPNs {
			if _, known := statuses[name]; !known {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return offerRestartForNewProfiles(base, missing)
}

// offerRestartForNewProfiles explains that names need a Kongtrol restart
// and runs it when the user agrees (or --yes); otherwise it says how.
func offerRestartForNewProfiles(base string, names []string) error {
	fmt.Println(tuiWarn(cf("cli.reload.new_profiles", strings.Join(names, ", "))))
	if !confirmAction(ct("cli.reload.ask_restart"), reloadYes, true) {
		fmt.Println(tuiInfo(ct("cli.reload.restart_hint")))
		return nil
	}
	return restartDaemonFn(base)
}

func init() {
	reloadCmd.Flags().StringVar(&reloadGroup, "group", "", ct("cli.reload.flag.group"))
	reloadCmd.Flags().BoolVar(&reloadPolicyOnly, "policy", false, ct("cli.reload.flag.policy"))
	reloadCmd.Flags().BoolVarP(&reloadYes, "yes", "y", false, ct("cli.reload.flag.yes"))
	reloadCmd.Flags().StringVar(&reloadTunnel, "tunnel", "", "")
	_ = reloadCmd.Flags().MarkHidden("tunnel")
	reloadCmd.MarkFlagsMutuallyExclusive("group", "tunnel", "policy")
	rootCmd.AddCommand(reloadCmd)
}
