package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

var upAll bool
var upGroup string
var upDryRun bool
var upFavorites bool

var upCmd = &cobra.Command{
	Use:   "up [profile...]",
	Short: "Connect one or more VPN profiles (or all with --all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		startedAll := time.Now()
		var (
			targets []string
			err     error
		)
		if upAll {
			targets = make([]string, 0, len(adapters))
			for name := range adapters {
				targets = append(targets, name)
			}
		} else {
			targets, err = resolveUpProfiles(args, upGroup, upFavorites)
			if err != nil {
				return err
			}
		}
		if upDryRun {
			return runUpDryRun(targets)
		}
		signalCtx := contextWithSignal()
		ctx, cancelCtx := context.WithCancel(signalCtx)
		defer cancelCtx()

		// Write PID so `kongtrol down` can stop this daemon.
		writePIDFile()
		defer removePIDFile()

		// Restore DNS on any exit (SIGTERM, panic path).
		defer func() {
			if dnsMgr != nil {
				dnsMgr.ForceRestore()
			}
		}()

		for _, name := range targets {
			startedProfile := time.Now()
			spin := newSpinner(fmt.Sprintf("Connecting %s", name))
			spin.Start()
			wasConnected := adapters[name].Status().Normalize() == vpn.StatusConnected
			connectCtx, cancelConnect := context.WithTimeout(ctx, 5*time.Minute)
			connectDone := make(chan error, 1)
			go func() {
				connectDone <- connectProfile(connectCtx, name)
			}()
			select {
			case err = <-connectDone:
			case <-ctx.Done():
				err = ctx.Err()
			}
			cancelConnect()
			spin.Stop()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					fmt.Println(tuiWarn(ct("cli.up.cancelled")))
					return nil
				}
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
					err = fmt.Errorf("%s", ct("cli.up.timeout"))
				}
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error() + "  " + paint(cDim, "("+time.Since(startedProfile).Round(time.Second).String()+")")))
				return fmt.Errorf("up %s: %w", name, err)
			}
			if wasConnected {
				fmt.Println(tuiInfo(paintBold(cBright, name) + "  " + paint(cDim, "already connected ("+time.Since(startedProfile).Round(time.Second).String()+")")))
			} else {
				fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, "connected in "+time.Since(startedProfile).Round(time.Second).String())))
			}
		}
		fmt.Println(tuiInfo(paint(cDim, cf("cli.up.connected_all", time.Since(startedAll).Round(time.Second)))))

		// Keep kill switch synchronized to active protected profiles.
		_ = applyKillSwitchState()
		if ks != nil {
			defer func() { _ = ks.Disable() }()
		}

		// Start watchdog after all profiles are up.
		if watchdog != nil {
			watchdog.Start(ctx)
			defer watchdog.Stop()
		}
		if scheduler != nil && cfg.Monitor.Scheduler.Enabled {
			scheduler.Start(ctx)
			defer scheduler.Stop()
		}

		// Start background DNS resolver for domain-based split tunneling.
		if policyResolver != nil {
			policyResolver.Start(ctx)
			defer policyResolver.Stop()
		}
		if splitDNSMgr != nil && cfg.Monitor.SplitDNS.Enabled {
			splitDNSMgr.Start(ctx)
			defer splitDNSMgr.Stop()
		}
		startHistoryPersistence(ctx)

		// Start background leak detection.
		if leak != nil {
			leak.Start(ctx, func(result security.LeakResult) {
				if result.HasLeak {
					if col != nil {
						for name, a := range adapters {
							if a.Status().Normalize() == vpn.StatusConnected {
								col.RecordLeak(name)
							}
						}
					}
					emitAlert("ERROR", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
					logAudit("SECURITY", "security.leak", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
				}
			})
			defer leak.Stop()
		}

		// Start the API server / dashboard alongside the daemon.
		var dashURL string
		srv := buildAPIServer(cancelCtx)
		if err := srv.Start(); err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.up.warn.dashboard_server", err)))
		} else {
			dashURL = srv.Addr()
			defer func() {
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutCtx)
			}()
		}

		// Block until signal — show live daemon view on interactive terminals.
		runUpTUI(ctx, cancelCtx, adapters, ks, dnsMgr, dashURL, true)
		return nil
	},
}

func init() {
	upCmd.Flags().BoolVar(&upAll, "all", false, ct("cli.up.flag.all"))
	upCmd.Flags().StringVar(&upGroup, "group", "", ct("cli.up.flag.group"))
	upCmd.Flags().BoolVar(&upDryRun, "dry-run", false, ct("cli.up.dry_run_help"))
	upCmd.Flags().BoolVar(&upFavorites, "fav", false, ct("cli.up.flag.fav"))
}
