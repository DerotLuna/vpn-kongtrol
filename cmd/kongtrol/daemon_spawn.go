package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// The spawn mechanics (lock, detached start, startup wait, log tail) live
// in internal/daemonctl so the tray starts the daemon exactly like `up`.
// This file keeps the CLI-specific parts: the daemon's command line and
// the failure rendering, plus test seams.

const (
	// spawnLockStale is how old a spawn lock may get before it is treated
	// as left behind by a crashed spawner and taken over.
	spawnLockStale = daemonctl.SpawnLockStale
)

// Indirections so the spawn decision logic is testable without starting
// real processes.
var (
	spawnDaemonFn     = spawnDaemon
	daemonProbeTick   = daemonctl.DefaultProbeTick
	daemonStartWaitFn = daemonctl.DefaultStartTimeout
	acquireSpawnLock  = daemonctl.AcquireSpawnLock
	readLogTail       = daemonctl.ReadLogTail
)

// errDaemonStart is a daemon that died or never answered during startup.
type errDaemonStart = daemonctl.StartError

// daemonLogPath returns ~/.kongtrol/daemon.log, where the detached daemon's
// stdout/stderr go.
func daemonLogPath() string { return daemonctl.LogPath() }

// spawnLockPath guards the probe-then-spawn window so two viewers racing
// each other don't both start a daemon.
func spawnLockPath() string { return daemonctl.LockPath() }

// ensureDaemon makes sure a daemon answers at base, spawning a detached
// one when none does. started reports whether this call spawned it.
func ensureDaemon(ctx context.Context, base string) (started bool, err error) {
	sp := daemonctl.Spawner{
		Probe:        func() bool { return daemonProbeFn(base) },
		Base:         base,
		Spawn:        spawnDaemonFn,
		LogPath:      daemonLogPath(),
		LockPath:     spawnLockPath(),
		StartTimeout: daemonStartWaitFn,
		ProbeTick:    daemonProbeTick,
		Lang:         cliLang(),
	}
	return sp.Ensure(ctx)
}

// spawnDaemon re-executes this binary as `kongtrol daemon`, fully detached
// from the calling terminal, with stdout and stderr appended to logPath.
func spawnDaemon(logPath string) (<-chan error, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return daemonctl.Spawn(exe, daemonArgs(), logPath, cliLang())
}

// daemonArgs builds the daemon's command line. Profiles are not passed:
// `up` connects them through the API once the daemon answers, so it can
// report each one's result. The resolved config path is passed as an
// absolute path because the daemon runs from the home directory.
func daemonArgs() []string {
	args := []string{"daemon", "--plain"}
	if activeCfgPath != "" {
		path := activeCfgPath
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		args = append(args, "--config", path)
	}
	if alertBell {
		args = append(args, "--alert-bell")
	}
	return args
}

// printDaemonStartFailure shows why a spawned daemon didn't come up,
// framed by translated header/footer around its own log lines.
func printDaemonStartFailure(err error) {
	var startErr *errDaemonStart
	if !errors.As(err, &startErr) {
		return
	}
	fmt.Fprintln(os.Stderr, tuiErr(cf("cli.daemon.up.start_failed", startErr.Cause)))
	if len(startErr.LogTail) == 0 {
		fmt.Fprintln(os.Stderr, paint(cDim, cf("cli.daemon.up.log_empty", daemonLogPath())))
		return
	}
	fmt.Fprintln(os.Stderr, paint(cDim, cf("cli.daemon.up.log_tail", daemonLogPath())))
	for _, line := range startErr.LogTail {
		fmt.Fprintln(os.Stderr, paint(cDim, "  │ ")+line)
	}
}
