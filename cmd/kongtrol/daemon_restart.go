package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Daemon restart: the adapters map is built once at boot (see loadConfig),
// so a new or edited VPN profile, and leaving first-run setup mode, need a
// fresh daemon process. POST /api/v1/restart (dashboard) and
// `kongtrol restart` (CLI) both end here: the running daemon starts its
// replacement with --restart and the tunnels that were up, then stops like
// on `kongtrol down`. The replacement waits until the old API is gone
// before binding the same address.

// daemonRestartWait is the hidden `daemon --restart` flag.
var daemonRestartWait bool

const (
	// restartOldGoneTimeout bounds how long a replacement waits for the old
	// daemon to disconnect its tunnels and stop.
	restartOldGoneTimeout = 2 * time.Minute
	restartPollEvery      = 250 * time.Millisecond
)

// Test seams.
var (
	spawnReplacementFn = daemonctl.Spawn
	restartSleep       = time.Sleep
)

func init() {
	daemonCmd.Flags().BoolVar(&daemonRestartWait, "restart", false, "")
	_ = daemonCmd.Flags().MarkHidden("restart")
}

// connectedProfiles lists the profiles whose tunnel is up, sorted.
func connectedProfiles() []string {
	var names []string
	for name, a := range adapters {
		if a.Status().Normalize() == vpn.StatusConnected {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// replacementArgs is the replacement daemon's command line.
func replacementArgs(targets []string) []string {
	args := daemonArgs()
	args = append(args, "--restart")
	return append(args, targets...)
}

// restartDaemon starts the replacement daemon; the API server then stops
// this one. The error is translated and carries a "what to do" line.
func restartDaemon() error {
	exe, err := currentExecutable()
	if err == nil {
		// In setup mode nothing is connected and activeCfgPath is the
		// default path the first profile was just written to.
		args := replacementArgs(connectedProfiles())
		_, err = spawnReplacementFn(exe, args, daemonLogPath(), cliLang())
	}
	if err != nil {
		return errors.New(cf("cli.restart.spawn_failed", err) + " · " + ct("cli.restart.spawn_failed_hint"))
	}
	logAudit("INFO", "daemon.restart", "", "restart requested")
	return nil
}

// waitDaemonGone polls base until no daemon answers there or timeout.
func waitDaemonGone(base string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for daemonProbeFn(base) {
		if time.Now().After(deadline) {
			return false
		}
		restartSleep(restartPollEvery)
	}
	return true
}

// knownTargets drops profiles a replacement daemon no longer has (deleted
// while the old daemon ran) instead of refusing to start.
func knownTargets(targets []string) []string {
	out := targets[:0]
	for _, name := range targets {
		if _, ok := adapters[name]; ok {
			out = append(out, name)
			continue
		}
		fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.restart.profile_gone", name)))
	}
	return out
}

// prepareReplacement runs at the start of a `daemon --restart`: it holds
// the spawn lock so clients wait instead of starting a daemon of their own,
// waits for the old daemon to stop, and returns the lock's release func.
func prepareReplacement(base string, targets []string) (release func(), kept []string, err error) {
	lockRelease, _ := acquireSpawnLock(spawnLockPath())
	// Idempotent: called once the API answers and again on exit; a second
	// call must not remove a lock another client took meanwhile.
	release = sync.OnceFunc(lockRelease)
	if !waitDaemonGone(base, restartOldGoneTimeout) {
		release()
		return func() {}, nil, fmt.Errorf("%s", ct("cli.restart.old_still_running"))
	}
	return release, knownTargets(targets), nil
}

// restartCmd is `kongtrol restart`: the CLI side of the dashboard's
// "Reiniciar Kongtrol" button. Unlike the dashboard (where the daemon starts
// its own replacement), the CLI stops the daemon and starts the new one from
// its own executable: after a package-manager upgrade the new `kongtrol`
// may live at a different path (Homebrew Cellar, Scoop version dir) than
// the binary the running daemon was started from.
var restartCmd = &cobra.Command{
	Use:  "restart",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		base := daemonAPIBase()
		if !probeDaemonAPI(base) {
			return fmt.Errorf("%s", ct("cli.reload.error.no_daemon"))
		}
		return restartDaemonFn(base)
	},
}

// restartDaemonFn is the restart path (test seam): `kongtrol restart`,
// `reload` when a new profile needs it and `init` → "connect now".
var restartDaemonFn = restartRunningDaemon

// restartRunningDaemon stops the daemon answering at base and starts a new
// one from this executable that reconnects the tunnels that were up.
func restartRunningDaemon(base string) error {
	c := daemonClient(base)
	statuses, err := c.Statuses()
	if err != nil {
		return err
	}
	var targets []string
	for name, st := range statuses {
		if st == vpn.StatusConnected {
			targets = append(targets, name)
		}
	}
	sort.Strings(targets)
	fmt.Println(tuiInfo(ct("cli.restart.requested")))
	if err := c.Shutdown(); err != nil {
		return err
	}
	if !waitDaemonGone(base, restartOldGoneTimeout) {
		return fmt.Errorf("%s", ct("cli.restart.timeout"))
	}
	spawn := func(logPath string) (<-chan error, error) {
		exe, err := currentExecutable()
		if err != nil {
			return nil, err
		}
		return spawnReplacementFn(exe, replacementArgs(targets), logPath, cliLang())
	}
	sp := daemonctl.Spawner{
		Probe:    func() bool { return daemonProbeFn(base) },
		Spawn:    spawn,
		LogPath:  daemonLogPath(),
		LockPath: spawnLockPath(),
		Lang:     cliLang(),
	}
	if _, err := sp.Ensure(context.Background()); err != nil {
		printDaemonStartFailure(err)
		return fmt.Errorf("%s", ct("cli.restart.timeout"))
	}
	fmt.Println(tuiOK(ct("cli.restart.done")))
	return nil
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
