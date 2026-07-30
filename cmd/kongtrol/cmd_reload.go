package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ── reload ───────────────────────────────────────────────────────────────────
//
// kongtrol reload picks up a hand-edited kongtrol.yaml in an already-running
// `kongtrol up` daemon, without restarting the whole process. It always talks
// to the running daemon's embedded API — the same well-known local address
// (daemonAPIBase) and probe/reachability check (probeDaemonAPI) that
// `status --watch` uses to decide whether to proxy actions to a real daemon
// instead of running them in-process. There is no local fallback here: unlike
// `up`/`down` (whose adapters largely proxy straight through to OS-level VPN
// client state, so a throwaway in-process instance still works), the entire
// point of `reload` is to update the *running* daemon's in-memory config,
// policy engine, watchdog, and DNS manager — state that only exists inside
// that one process — so it fails clearly if no daemon is reachable rather
// than silently reloading a copy nobody is using.
var reloadGroup string
var reloadTunnel string
var reloadPolicyOnly bool

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload kongtrol.yaml from disk into the running daemon (policies, groups, and/or a single tunnel)",
	RunE: func(cmd *cobra.Command, args []string) error {
		base := daemonAPIBase()
		if !probeDaemonAPI(base) {
			return fmt.Errorf("%s", ct("cli.reload.error.no_daemon"))
		}

		if err := daemonReloadPolicy(base); err != nil {
			return fmt.Errorf("%s: %w", ct("cli.reload.error.policy"), err)
		}
		fmt.Println(tuiOK(ct("cli.reload.policy_reloaded")))

		if reloadPolicyOnly {
			return nil
		}

		if reloadTunnel != "" {
			result, err := daemonReloadTunnel(base, reloadTunnel)
			if err != nil {
				return fmt.Errorf("%s: %w", cf("cli.reload.error.tunnel", reloadTunnel), err)
			}
			if result.RestartRequired {
				fmt.Println(tuiWarn(cf("cli.reload.tunnel_restart_required", reloadTunnel)))
				return nil
			}
			if len(result.Restarted) == 0 {
				fmt.Println(tuiOK(cf("cli.reload.tunnel_skipped", reloadTunnel)))
				return nil
			}
			fmt.Println(tuiOK(cf("cli.reload.tunnel_reloaded", reloadTunnel)))
			return nil
		}

		groups := []string{}
		if reloadGroup != "" {
			groups = []string{reloadGroup}
		} else {
			names, err := daemonGroupNames(base)
			if err != nil {
				return fmt.Errorf("%s: %w", ct("cli.reload.error.list_groups"), err)
			}
			groups = names
		}

		for _, g := range groups {
			result, err := daemonReloadGroup(base, g)
			if err != nil {
				fmt.Println(tuiErr(cf("cli.reload.group_failed", g, err)))
				continue
			}
			if result.RestartRequired {
				fmt.Println(tuiWarn(cf("cli.reload.group_restart_required", g, strings.Join(result.MissingProfiles, ", "))))
				continue
			}
			fmt.Println(tuiOK(cf("cli.reload.group_reloaded", g)))
		}
		return nil
	},
}

func init() {
	reloadCmd.Flags().StringVar(&reloadGroup, "group", "", ct("cli.reload.flag.group"))
	reloadCmd.Flags().StringVar(&reloadTunnel, "tunnel", "", ct("cli.reload.flag.tunnel"))
	reloadCmd.Flags().BoolVar(&reloadPolicyOnly, "policy", false, ct("cli.reload.flag.policy"))
	reloadCmd.MarkFlagsMutuallyExclusive("group", "tunnel", "policy")
	rootCmd.AddCommand(reloadCmd)
}
