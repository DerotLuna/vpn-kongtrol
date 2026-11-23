package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

var upAll bool
var upGroup string
var upDryRun bool
var upFavorites bool

// upCmd is a client of the background daemon (`kongtrol daemon`, see
// daemon_cmd.go), the Docker/Tailscale model: it makes sure a daemon is
// running — spawning a detached one when none answers — connects the
// requested profiles through the daemon's API, and then attaches the live
// view. Quitting the view never stops the daemon or its tunnels; only
// `kongtrol down` does, and no terminal has to stay open.
var upCmd = &cobra.Command{
	Use: "up [profile...]",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			targets []string
			err     error
		)
		if upAll {
			targets = adapters.Names()
		} else {
			targets, err = resolveUpProfiles(args, upGroup, upFavorites)
			if err != nil {
				return err
			}
		}
		if upDryRun {
			return runUpDryRun(targets)
		}
		return runUp(contextWithSignal(), targets)
	},
}

func init() {
	upCmd.Flags().BoolVar(&upAll, "all", false, ct("cli.up.flag.all"))
	upCmd.Flags().StringVar(&upGroup, "group", "", ct("cli.up.flag.group"))
	upCmd.Flags().BoolVar(&upDryRun, "dry-run", false, ct("cli.up.dry_run_help"))
	upCmd.Flags().BoolVar(&upFavorites, "fav", false, ct("cli.up.flag.fav"))
	upCmd.Flags().BoolVar(&alertBell, "alert-bell", false, ct("cli.flag.alert_bell"))
}

// Indirections so runUp is testable without a terminal or a real daemon.
var (
	attachViewerFn = func(ctx context.Context, cancel context.CancelFunc, base string) {
		runUpTUI(ctx, cancel, adapters.Snapshot(), ks, dnsMgr, base, false)
	}
	upIsTerminalFn = isTerminal
	// upPollInterval is how often a pending connect's state is re-read.
	upPollInterval = time.Second
	// upConnectTimeout matches the daemon's own per-connect bound.
	upConnectTimeout = 5 * time.Minute
	// upNoProgressGrace is how long a profile may still read
	// "disconnected" after its connect was accepted before that is taken
	// as a failure; it covers one collector poll (5s) of snapshot lag.
	upNoProgressGrace = 15 * time.Second
)

// runUp ensures the daemon, connects targets through it with a spinner and
// result line per profile (as `up` always showed), then attaches the live
// view on interactive terminals, or prints a hint and exits otherwise.
func runUp(ctx context.Context, targets []string) error {
	startedAll := time.Now()
	base := daemonAPIBase()

	spin := newSpinner(ct("cli.daemon.up.starting"))
	spin.Start()
	started, err := ensureDaemon(ctx, base)
	spin.Stop()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println(tuiWarn(ct("cli.up.cancelled")))
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
	} else {
		fmt.Println(tuiInfo(ct("cli.daemon.up.attached")))
	}

	for _, name := range targets {
		startedProfile := time.Now()
		baseLabel := cf("cli.daemon.up.connecting", name)
		spin := newSpinner(baseLabel)
		spin.Start()
		// Provider adapters (attach mode) explain each step: waiting for
		// the user in the app replaces the spinner text; how the tunnel
		// came up replaces the generic "connected" line.
		var attach *vpn.AttachState
		onState := func(m monitor.TunnelMetrics) {
			attach = m.Attach
			if l := attachSpinnerLabel(name, m.Attach, "attach.hint.cancel"); l != "" {
				spin.SetLabel(l)
			} else {
				spin.SetLabel(baseLabel)
			}
		}
		already, err := connectViaDaemonProgress(ctx, base, name, onState)
		spin.Stop()
		// First run: the helper is missing or outdated. On an interactive
		// terminal offer to install it (one OS permission prompt) and try
		// this profile once more; otherwise the error below says how.
		if need := connectHelperNeed(err); need != helperReady && canPrompt() {
			if offerHelperInstall(need, false) {
				spin = newSpinner(baseLabel)
				spin.Start()
				already, err = connectViaDaemonProgress(ctx, base, name, onState)
				spin.Stop()
			}
		}
		elapsed := time.Since(startedProfile).Round(time.Second)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println(tuiWarn(ct("cli.up.cancelled")))
				return nil
			}
			// A refusal with its "what to do" line (e.g. another VPN
			// already carries all traffic, ADR-0099) shows both.
			err = remoteUserErr(err)
			fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error() + "  " + paint(cDim, "("+elapsed.String()+")")))
			fmt.Println(tuiInfo(paint(cDim, ct("cli.up.failed_hint"))))
			return alreadyReported(fmt.Errorf("up %s: %w", name, err))
		}
		if already {
			fmt.Println(tuiInfo(paintBold(cBright, name) + "  " + paint(cDim, cf("cli.daemon.up.already_connected", elapsed))))
		} else if line := attachDoneLine(attach); line != "" {
			fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, line+"  ("+elapsed.String()+")")))
		} else {
			fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, cf("cli.daemon.up.connected_in", elapsed))))
		}
		if !already {
			if line := upRoutesLine(base, name); line != "" {
				fmt.Println(tuiInfo(paint(cDim, line)))
			}
		}
	}
	if len(targets) > 0 {
		fmt.Println(tuiInfo(paint(cDim, cf("cli.up.connected_all", time.Since(startedAll).Round(time.Second)))))
		// After the first VPN: reconnect at start? Asked once (ADR-0101).
		offerRestoreAtStart(daemonClient(base))
	}

	if !upIsTerminalFn() {
		fmt.Println(tuiInfo(paint(cDim, ct("cli.daemon.up.hint"))))
		return nil
	}
	viewCtx, cancelView := context.WithCancel(ctx)
	defer cancelView()
	attachViewerFn(viewCtx, cancelView, base)
	return nil
}

// connectViaDaemon asks the daemon to connect name and follows the tunnel's
// state through the daemon's API until it is connected, fails, or times
// out. Cancelling ctx (Ctrl+C) aborts the daemon-side connect too.
func connectViaDaemon(ctx context.Context, base, name string) (alreadyConnected bool, err error) {
	return connectViaDaemonProgress(ctx, base, name, nil)
}

// connectViaDaemonProgress is connectViaDaemon that hands every state read
// of the tunnel to onState (nil to skip). While a provider adapter waits
// for the user in its app (attach phase waiting_user), `up`'s own timeout
// is suspended: the adapter bounds that wait with wait_timeout.
func connectViaDaemonProgress(ctx context.Context, base, name string, onState func(monitor.TunnelMetrics)) (alreadyConnected bool, err error) {
	already, err := daemonConnectFn(base, name)
	if err != nil || already {
		return already, err
	}

	start := time.Now()
	tick := time.NewTicker(upPollInterval)
	defer tick.Stop()
	seenProgress := false
	for {
		select {
		case <-ctx.Done():
			_ = daemonCancelConnectFn(base, name)
			return false, ctx.Err()
		case <-tick.C:
		}
		states, err := daemonTunnelStatesFn(base)
		if err != nil {
			// Transient read failure; the timeout still bounds this.
			states = nil
		}
		state, ok := states[name]
		if ok && state.Status == vpn.StatusConnecting {
			seenProgress = true
		}
		if ok && onState != nil {
			onState(state)
		}
		done, err := upConnectDecision(state, ok, seenProgress, time.Since(start))
		if done {
			return false, err
		}
		if ok && attachWaitingUser(state.Attach) {
			continue
		}
		if time.Since(start) > upConnectTimeout {
			_ = daemonCancelConnectFn(base, name)
			return false, fmt.Errorf("%s", ct("cli.up.timeout"))
		}
	}
}

// upConnectError is a failed connect as the daemon reported it: the
// translated line and the guidance key it came with ("" when none).
type upConnectError struct{ msg, key string }

func (e *upConnectError) Error() string { return e.msg }

// connectHelperNeed says whether err failed because the privileged helper
// is missing or outdated (a tunnel's last_error_key, or the daemon's
// error_key on the connect request itself).
func connectHelperNeed(err error) helperNeed {
	var ce *upConnectError
	if errors.As(err, &ce) {
		return helperNeedFromKey(ce.key)
	}
	var keyed interface{ ErrorKey() string }
	if errors.As(err, &keyed) {
		return helperNeedFromKey(keyed.ErrorKey())
	}
	return helperReady
}

// upConnectDecision decides, from one read of the daemon's view of a
// tunnel after its connect was accepted, whether `up` is done waiting and
// with which result. The daemon clears a tunnel's last error when the
// connect starts, so any LastError seen here belongs to this attempt and
// is shown verbatim instead of only pointing at the daemon log.
func upConnectDecision(state monitor.TunnelMetrics, known, seenProgress bool, elapsed time.Duration) (done bool, err error) {
	if !known {
		return false, nil
	}
	failed := func() error {
		if state.LastError != "" {
			return &upConnectError{
				msg: cf("cli.daemon.up.connect_failed_reason", tunnelErrorText(state)),
				key: state.LastErrorKey,
			}
		}
		return fmt.Errorf("%s", ct("cli.daemon.up.connect_failed"))
	}
	switch state.Status {
	case vpn.StatusConnected:
		return true, nil
	case vpn.StatusError:
		return true, failed()
	case vpn.StatusConnecting:
		return false, nil
	default:
		if state.LastError != "" || seenProgress || elapsed > upNoProgressGrace {
			return true, failed()
		}
		return false, nil
	}
}
