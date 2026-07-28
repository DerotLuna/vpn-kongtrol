package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
)

// ── down ─────────────────────────────────────────────────────────────────────

var downAll bool
var downGroup string

var downCmd = &cobra.Command{
	Use:   "down [profile...]",
	Short: "Disconnect one or more VPN profiles (or a group with --group)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := contextWithSignal()
		disconnectWithSpinner := func(name string) error {
			spin := newSpinner(cf("cli.down.disconnecting", name))
			spin.Start()
			err := disconnectProfile(ctx, name)
			spin.Stop()
			if err != nil {
				fmt.Println(tuiErr(paintBold(cBright, name) + "  " + err.Error()))
				return err
			}
			fmt.Println(tuiOK(paintBold(cBright, name) + "  " + paint(cDim, ct("cli.down.disconnected"))))
			return nil
		}

		if downAll {
			names := make([]string, 0, len(adapters))
			for name := range adapters {
				names = append(names, name)
			}
			disconnectAllConcurrently(ctx, names, disconnectProfile)
			stopDaemon()
			return nil
		}

		targets, err := resolveProfiles(args, downGroup)
		if err != nil {
			return err
		}
		for _, name := range targets {
			if err := disconnectWithSpinner(name); err != nil {
				return fmt.Errorf("down %s: %w", name, err)
			}
		}
		stopDaemon()
		return nil
	},
}

func init() {
	downCmd.Flags().BoolVar(&downAll, "all", false, ct("cli.down.flag.all"))
	downCmd.Flags().StringVar(&downGroup, "group", "", ct("cli.down.flag.group"))
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
			lines = append(lines, tuiErr(paintBold(cBright, r.name)+"  "+r.err.Error()))
		} else {
			lines = append(lines, tuiOK(paintBold(cBright, r.name)+"  "+paint(cDim, ct("cli.down.disconnected"))))
		}
	}
	agg.Stop()
	for _, l := range lines {
		fmt.Println(l)
	}
}
