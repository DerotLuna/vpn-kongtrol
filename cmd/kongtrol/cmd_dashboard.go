package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// ── dashboard ─────────────────────────────────────────────────────────────────

// dashboardCmd opens the dashboard of the shared daemon, like the tray's
// "Abrir dashboard": it starts `kongtrol daemon` when none answers (the
// same Spawner `up` uses) and only opens the browser. It never serves the
// dashboard itself, so closing this terminal changes nothing.
var dashboardCmd = &cobra.Command{
	Use:  "dashboard",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		base := daemonAPIBase()
		spin := newSpinner(ct("cli.daemon.up.starting"))
		spin.Start()
		started, err := ensureDaemon(contextWithSignal(), base)
		spin.Stop()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			var startErr *errDaemonStart
			if errors.As(err, &startErr) {
				printDaemonStartFailure(err)
				fmt.Println(tuiInfo(paint(cDim, ct("cli.up.failed_hint"))))
				return alreadyReported(err)
			}
			return err
		}
		if started {
			fmt.Println(tuiOK(ct("cli.daemon.up.started")))
		}
		fmt.Println(tuiOK(styleBright.Render(ct("cli.dashboard.running")) + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(base)))
		fmt.Println(paint(cDim, "  "+ct("cli.dashboard.opening")))
		if err := openBrowserFn(base); err != nil {
			fmt.Println(tuiWarn(cf("cli.dashboard.open_failed", err)))
		}
		fmt.Println(tuiInfo(paint(cDim, ct("cli.dashboard.keeps_running"))))
		return nil
	},
}

// openBrowserFn is a test seam.
var openBrowserFn = openBrowser
