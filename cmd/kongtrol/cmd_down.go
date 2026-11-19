package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── down ─────────────────────────────────────────────────────────────────────

var downAll bool
var downGroup string

// downRemember is the hidden `down --all --remember` an installer runs
// before it replaces the files: the profiles that are connected are saved
// for `kongtrol resume` (ADR-0050).
var downRemember bool

var downCmd = &cobra.Command{
	Use: "down [profile...]",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDown(contextWithSignal(), args)
	},
}

// stopDaemonFn is stopDaemon behind an indirection so runDown's
// stop-or-keep decision can be tested without a real daemon process.
var stopDaemonFn = stopDaemon

// runDown disconnects the requested profiles and then decides whether the
// background daemon should be stopped.
//
// When a daemon's API is reachable, every disconnect is routed through it:
// the daemon owns the real adapter instances (and, for process-owning
// adapters like openvpn, the child process), while this CLI process only
// has fresh, never-connected copies whose Disconnect would be a no-op and
// leave the tunnel orphaned. Without a reachable daemon the local adapters
// are used, as before.
//
// Because stopping the daemon now tears down every tunnel it still holds,
// a targeted `down <profile>` against a live daemon only stops it once no
// other profile remains active there; --all, and the no-daemon path, keep
// the previous "always stop the daemon" behavior.
func runDown(ctx context.Context, args []string) error {
	base := daemonAPIBase()
	viaDaemon := daemonProbeFn(base)
	rawDisconnect := disconnectProfile
	readState := localTunnelState
	if viaDaemon {
		rawDisconnect = func(_ context.Context, name string) error {
			return daemonDisconnectFn(base, name)
		}
		readState = func(name string) (monitor.TunnelMetrics, bool) {
			states, err := daemonTunnelStatesFn(base)
			if err != nil {
				return monitor.TunnelMetrics{}, false
			}
			st, ok := states[name]
			return st, ok
		}
	}
	// A provider app without a CLI (attach mode) can't be disconnected by
	// Kongtrol: the user disconnects in the app and `down` waits for the
	// tunnel to drop, showing what to do.
	var activeSpin *spinner
	disconnect := func(ctx context.Context, name string) error {
		err := rawDisconnect(ctx, name)
		if !errors.Is(err, vpn.ErrManualDisconnect) {
			return err
		}
		return waitManualDown(ctx, name, readState, activeSpin)
	}

	if downAll {
		if downRemember {
			rememberConnected(base, viaDaemon)
		}
		names := make([]string, 0, adapters.Len())
		for _, name := range adapters.Names() {
			names = append(names, name)
		}
		disconnectAllConcurrently(ctx, names, disconnect)
		stopDaemonFn()
		return nil
	}

	targets, err := resolveProfiles("down", args, downGroup)
	if err != nil {
		return err
	}
	for _, name := range targets {
		spin := newSpinner(cf("cli.down.disconnecting", name))
		spin.Start()
		activeSpin = spin
		err := disconnect(ctx, name)
		activeSpin = nil
		spin.Stop()
		if err != nil {
			if line, ok := errorGuidance(err); ok {
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + line))
			} else {
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error()))
				fmt.Println(tuiInfo(paint(cDim, ct("cli.down.failed_hint"))))
			}
			return alreadyReported(fmt.Errorf("down %s: %w", name, err))
		}
		fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, ct("cli.down.disconnected"))))
	}

	if viaDaemon {
		if remaining := daemonRemainingAfter(base, targets); len(remaining) > 0 {
			fmt.Println(tuiInfo(paint(cDim, cf("cli.daemon.down.kept_running", strings.Join(remaining, ", ")))))
			return nil
		}
	}
	stopDaemonFn()
	return nil
}

// rememberConnected saves the profiles the daemon reports connected for
// `kongtrol resume`. With no daemon (or no answer) nothing was up through
// Kongtrol, so any previous file is removed instead. A failure only costs
// the automatic reconnect, never the stop itself.
func rememberConnected(base string, viaDaemon bool) {
	var up []string
	if viaDaemon {
		if statuses, err := daemonTunnelStatusesFn(base); err == nil {
			up = connectedNames(statuses)
		}
	}
	_ = writeResumeFn(daemonctl.ResumePath(), up, time.Now())
}

// connectedNames lists the connected profiles in statuses, sorted.
func connectedNames(statuses map[string]vpn.Status) []string {
	var names []string
	for name, st := range statuses {
		if st.Normalize() == vpn.StatusConnected {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// writeResumeFn is a test seam over daemonctl.WriteResume.
var writeResumeFn = daemonctl.WriteResume

// daemonRemainingAfter returns the profiles the daemon still reports active
// once the just-disconnected ones are discounted (its snapshot can lag a
// synchronous disconnect by one collector poll). If the daemon's state
// can't be read, it returns nil so the caller falls back to stopping the
// daemon, matching the pre-existing behavior.
func daemonRemainingAfter(base string, disconnected []string) []string {
	active, err := daemonActiveProfilesFn(base)
	if err != nil {
		return nil
	}
	done := make(map[string]bool, len(disconnected))
	for _, name := range disconnected {
		done[name] = true
	}
	var remaining []string
	for _, name := range active {
		if !done[name] {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	return remaining
}

func init() {
	downCmd.Flags().BoolVar(&downAll, "all", false, ct("cli.down.flag.all"))
	downCmd.Flags().StringVar(&downGroup, "group", "", ct("cli.down.flag.group"))
	downCmd.Flags().BoolVar(&downRemember, "remember", false, "")
	_ = downCmd.Flags().MarkHidden("remember")
}

// disconnectAllConcurrently disconnects every named profile in parallel
// instead of one at a time — with several active tunnels the sequential
// version added up to real wait time for no benefit, since each profile's
// adapter is independent.
//
// Per-profile live spinners (used by the single/multi-target path above)
// aren't used here: they animate by repeatedly overwriting the current
// terminal line via \r, and multiple concurrent spinners racing on that
// same line would corrupt the display. Instead this shows one aggregate
// spinner while all disconnects run, then prints each result once
// everything has settled. Matches the original --all semantics: every
// profile is attempted and its error (if any) is reported, but a single
// profile failing doesn't stop the others or fail the command.
func disconnectAllConcurrently(ctx context.Context, names []string, disconnect func(context.Context, string) error) {
	if len(names) == 0 {
		return
	}

	agg := newSpinner(cf("cli.down.disconnecting_all", len(names)))
	agg.Start()

	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- result{name: name, err: disconnect(ctx, name)}
		}(name)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	lines := make([]string, 0, len(names))
	for r := range results {
		if r.err != nil {
			msg := r.err.Error()
			if line, ok := errorGuidance(r.err); ok {
				msg = line
			}
			lines = append(lines, tuiErr(paintBold(cBright, r.name)+"  "+msg))
		} else {
			lines = append(lines, tuiOK(paintBold(cBright, r.name)+"  "+paint(cDim, ct("cli.down.disconnected"))))
		}
	}
	agg.Stop()
	for _, l := range lines {
		fmt.Println(l)
	}
}
