package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// resumeCmd is the hidden `kongtrol resume` installers run once the new
// files are in place (ADR-0050): it reconnects the profiles that
// `down --all --remember` saved before the update. It reuses the restart
// path: with no daemon it starts `daemon --restart <profiles>` (the same
// replacement daemon `kongtrol restart` starts, which drops profiles that
// no longer exist); with a daemon already running (the tray was faster)
// it asks that daemon to connect them. Provider apps follow the usual
// attach rules of `up`: attach when their tunnel is still up, otherwise the
// normal connect path. Without a saved list it does nothing.
var resumeCmd = &cobra.Command{
	Use:    "resume",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runResume(context.Background())
	},
}

// Test seams (loadConfigFn is shared with finishInit).
var (
	takeResumeFn  = daemonctl.TakeResume
	ensureResumed = ensureReplacementDaemon
)

func runResume(ctx context.Context) error {
	saved, err := takeResumeFn(daemonctl.ResumePath(), time.Now())
	if err != nil {
		return fmt.Errorf("%s · %s", cf("cli.resume.read_failed", err), ct("cli.resume.failed_hint"))
	}
	if len(saved) == 0 {
		resumeSay(tuiInfo(ct("cli.resume.none")))
		return nil
	}
	// Loaded here, not in PersistentPreRunE: a first install has no
	// kongtrol.yaml and nothing to resume, and must not fail.
	if err := loadConfigFn(); err != nil {
		return err
	}
	targets := resumeTargets(saved)
	if len(targets) == 0 {
		return nil
	}
	resumeSay(tuiInfo(cf("cli.resume.reconnecting", strings.Join(targets, ", "))))

	base := daemonAPIBase()
	if daemonProbeFn(base) {
		failed := 0
		for _, name := range targets {
			if _, err := daemonConnectFn(base, name); err != nil {
				failed++
				fmt.Fprintln(os.Stderr, tuiErr(cf("cli.resume.connect_failed", name, err)))
			}
		}
		if failed > 0 {
			return alreadyReported(fmt.Errorf("resume: %d failed", failed))
		}
		resumeSay(tuiOK(ct("cli.resume.started")))
		return nil
	}
	if err := ensureResumed(ctx, base, targets); err != nil {
		printDaemonStartFailure(err)
		return fmt.Errorf("%s · %s", ct("cli.resume.start_failed"), ct("cli.resume.failed_hint"))
	}
	resumeSay(tuiOK(ct("cli.resume.started")))
	return nil
}

// resumeTargets keeps the saved profiles that still exist in the config,
// in order, and says which ones are skipped.
func resumeTargets(saved []string) []string {
	var out []string
	for _, name := range saved {
		if _, ok := cfg.VPNs[name]; ok {
			out = append(out, name)
			continue
		}
		fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.restart.profile_gone", name)))
	}
	return out
}

// ensureReplacementDaemon starts `daemon --restart <targets>` from this
// executable, exactly like restartRunningDaemon does after stopping the old
// daemon, and waits until its API answers. The daemon then connects the
// targets itself.
func ensureReplacementDaemon(ctx context.Context, base string, targets []string) error {
	sp := daemonctl.Spawner{
		Probe: func() bool { return daemonProbeFn(base) },
		Spawn: func(logPath string) (<-chan error, error) {
			exe, err := currentExecutable()
			if err != nil {
				return nil, err
			}
			return spawnReplacementFn(exe, replacementArgs(targets), logPath, cliLang())
		},
		LogPath:  daemonLogPath(),
		LockPath: spawnLockPath(),
		Lang:     cliLang(),
	}
	_, err := sp.Ensure(ctx)
	return err
}

func resumeSay(line string) {
	if !outputQuiet {
		fmt.Println(line)
	}
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
