package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// ── dashboard ─────────────────────────────────────────────────────────────────

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web dashboard and open it in your browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, _, err := fetchDaemonSnapshot(); err == nil {
			addr := dashboardURL()
			fmt.Println(tuiOK(styleBright.Render(ct("cli.dashboard.running")) + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
			fmt.Println(paint(cDim, "  "+ct("cli.dashboard.opening")))
			if err := openBrowser(addr); err != nil {
				fmt.Println(tuiWarn(cf("cli.dashboard.open_failed", err)))
			}
			return nil
		}

		ctx, cancel := context.WithCancel(contextWithSignal())
		defer cancel()

		srv := buildAPIServer(cancel)
		if err := srv.Start(); err != nil {
			return fmt.Errorf("%s", cf("cli.dashboard.error_start", err))
		}
		addr := dashboardURL()
		fmt.Println(tuiOK(styleBright.Render(ct("cli.dashboard.running")) + "  " + paint(cDim, "→") + "  " + stylePrompt.Render(addr)))
		fmt.Println(paint(cDim, "  "+ct("cli.dashboard.opening")))
		openBrowser(addr)
		fmt.Println(paint(cDim, "  "+ct("cli.dashboard.ctrlc_stop")))
		fmt.Println()

		// Block until Ctrl+C or a POST /api/v1/shutdown.
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		return srv.Shutdown(shutCtx)
	},
}
