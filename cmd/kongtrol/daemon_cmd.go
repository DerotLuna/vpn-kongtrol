package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// daemonCmd is the headless daemon that owns every tunnel. It is hidden
// because users never run it directly: `kongtrol up` spawns it detached
// (see daemon_spawn.go) and `kongtrol down` stops it through its API. It
// is a separate command rather than an `up --foreground` flag so `up`
// stays a pure client with no daemon code path, and so a service manager
// (systemd, launchd, a Windows service wrapper) has one stable entrypoint.
var daemonCmd = &cobra.Command{
	Use:    "daemon [profile...]",
	Hidden: true,
	Args:   cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon(contextWithSignal(), args)
	},
}

func init() {
	daemonCmd.Short = ct("cli.daemon.short")
	// Passed on by up (daemonArgs); local so the deprecated global one
	// is not what the daemon parses.
	daemonCmd.Flags().BoolVar(&alertBell, "alert-bell", false, "")
	_ = daemonCmd.Flags().MarkHidden("alert-bell")
	rootCmd.AddCommand(daemonCmd)
}

// runDaemon runs the daemon until ctx is cancelled (SIGINT/SIGTERM) or a
// client POSTs /api/v1/shutdown. The API server is started first and is
// mandatory: it is the only way `up`, `down`, `status --watch` and the
// dashboard can find and drive this headless process, so a daemon whose API
// cannot bind exits with an error instead of running undetectable.
// Requested profiles are connected after the API is up, so `up` can detect
// the daemon immediately and follow each connect's progress through it.
func runDaemon(signalCtx context.Context, targets []string) error {
	base := daemonAPIBase()
	releaseRestartLock := func() {}
	if daemonRestartWait {
		release, kept, err := prepareReplacement(base, targets)
		if err != nil {
			return err
		}
		releaseRestartLock, targets = release, kept
		defer releaseRestartLock()
	}
	for _, name := range targets {
		if _, ok := adapters.Get(name); !ok {
			return fmt.Errorf("%s", cf("cli.error.unknown_profile", name))
		}
	}
	if daemonProbeFn(base) {
		return fmt.Errorf("%s", cf("cli.daemon.already_running", base))
	}

	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	fmt.Println(tuiInfo(cf("cli.daemon.starting", os.Getpid(), time.Now().Format(time.RFC3339))))

	srv := buildAPIServer(cancel)
	if err := srv.Start(); err != nil {
		return fmt.Errorf("%s", cf("cli.daemon.api_failed", base, err))
	}
	defer func() {
		shutCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = srv.Shutdown(shutCtx)
	}()
	// A replacement daemon answers now: clients may stop waiting.
	releaseRestartLock()
	if setupMode {
		fmt.Println(tuiInfo(cf("cli.daemon.setup_mode", activeCfgPath)))
	}

	// Only the daemon writes the PID file; `down` uses it as the kill
	// fallback and waits for its removal as the "fully stopped" signal.
	writePIDFile()
	defer removePIDFile()

	defer func() {
		if dnsMgr != nil {
			dnsMgr.ForceRestore()
		}
	}()
	// Safety net for every exit path; runs before DNS restore and PID
	// removal. The normal path below already disconnected everything.
	defer disconnectActiveOnShutdown(adapters.Snapshot(), disconnectProfile, shutdownDisconnectTimeout)

	stopServices := startDaemonServices(ctx)
	defer stopServices()
	auditDaemonStart()

	fmt.Println(tuiInfo(cf("cli.daemon.listening", base)))

	for _, name := range targets {
		if ctx.Err() != nil {
			break
		}
		started := time.Now()
		connectCtx, cancelConnect := context.WithTimeout(ctx, 5*time.Minute)
		err := connectProfile(connectCtx, name)
		cancelConnect()
		elapsed := time.Since(started).Round(time.Second)
		switch {
		case err == nil:
			fmt.Println(tuiOK(name + "  " + cf("cli.daemon.up.connected_in", elapsed)))
		case errors.Is(err, context.Canceled):
			fmt.Println(tuiWarn(name + "  " + ct("cli.up.cancelled")))
		default:
			fmt.Println(tuiErr(name + "  " + err.Error()))
		}
	}

	<-ctx.Done()
	fmt.Println(tuiInfo(ct("cli.daemon.stopping")))
	auditDaemonStop()

	// Disconnect while kill switch, watchdog and API are still alive
	// (their defers run after this), strictly before DNS restore and PID
	// file removal.
	disconnectActiveOnShutdown(adapters.Snapshot(), disconnectProfile, shutdownDisconnectTimeout)
	return nil
}

// startDaemonServices starts the background subsystems the daemon runs for
// its whole lifetime and returns a function that stops them in reverse
// order.
func startDaemonServices(ctx context.Context) func() {
	var stops []func()

	// Keep kill switch synchronized to active protected profiles.
	_ = applyKillSwitchState()
	if ks != nil {
		stops = append(stops, func() { _ = ks.Disable() })
	}
	// Split-tunnel kill switch + on_tunnel_down: block. The loop notices
	// unexpected drops; stopping the daemon (`down`) lifts every block.
	if killSwitchSvc != nil {
		go killSwitchSvc.Run(ctx, 5*time.Second)
		stops = append(stops, func() {
			if destBlocker != nil {
				_ = destBlocker.Set(nil)
			}
		})
	}
	// Notices for traffic of a policy whose tunnel is off.
	if policyWatch != nil {
		go policyWatch.Run(ctx, 15*time.Second)
	}
	if recentConns != nil {
		go recentConns.Run(ctx, 15*time.Second) // monitor.RecentEvery
	}
	if watchdog != nil {
		watchdog.Start(ctx)
		stops = append(stops, watchdog.Stop)
	}
	// The scheduler follows scheduler.enabled itself, also when it changes
	// later (applySchedulerConfig), so it is always started.
	if scheduler != nil {
		scheduler.Start(ctx)
		stops = append(stops, scheduler.Stop)
	}
	// Background DNS resolver for domain-based split tunneling.
	if policyResolver != nil {
		policyResolver.Start(ctx)
		stops = append(stops, policyResolver.Stop)
	}
	if splitDNSMgr != nil && cfg.Monitor.SplitDNS.Enabled {
		splitDNSMgr.Start(ctx)
		stops = append(stops, splitDNSMgr.Stop)
	}
	startHistoryPersistence(ctx)
	// Opt-in update notice: no request unless updates.check_enabled.
	startUpdateNotice(ctx)

	// Leak alerts from the on-demand public IP lookup (it never polls: it
	// runs on tunnel connect/disconnect and when the user asks).
	if leak != nil {
		leak.SetOnResult(func(result security.LeakResult) {
			if !result.HasLeak {
				return
			}
			if col != nil {
				for name, a := range adapters.Snapshot() {
					if a.Status().Normalize() == vpn.StatusConnected {
						col.RecordLeak(name)
					}
				}
			}
			emitAlert("ERROR", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
			logAudit("SECURITY", "security.leak", "", cf("cli.alert.leak_detected", result.Reason, result.PublicIP))
		})
		stops = append(stops, leak.Stop)
	}

	return func() {
		for i := len(stops) - 1; i >= 0; i-- {
			stops[i]()
		}
	}
}

var logsDaemon bool

func init() {
	// `kongtrol logs --daemon` shows the detached daemon's own output
	// (daemon.log) instead of the audit log. Hooked from here so the audit
	// log command itself stays untouched.
	logsCmd.Flags().BoolVar(&logsDaemon, "daemon", false, ct("cli.daemon.logs.flag"))
	auditRun := logsCmd.RunE
	logsCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if logsDaemon {
			return showDaemonLog(contextWithSignal(), daemonLogPath(), logsTail, logsFollow)
		}
		return auditRun(cmd, args)
	}
}

// showDaemonLog prints the last tail lines of path and, with follow, keeps
// printing whatever is appended until ctx ends.
func showDaemonLog(ctx context.Context, path string, tail int, follow bool) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s", cf("cli.daemon.logs.missing", path))
	}
	if tail <= 0 {
		tail = 100
	}
	for _, line := range readLogTail(path, 0, tail) {
		fmt.Println(line)
	}
	if !follow {
		return nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	offset := fi.Size()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		if fi.Size() < offset {
			offset = 0 // truncated or replaced
		}
		if fi.Size() == offset {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		if _, err := f.Seek(offset, io.SeekStart); err == nil {
			n, _ := io.Copy(os.Stdout, f)
			offset += n
		}
		_ = f.Close()
	}
}
