package daemonctl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

const (
	// DefaultStartTimeout bounds how long Ensure waits for a freshly
	// spawned daemon's API to answer.
	DefaultStartTimeout = 30 * time.Second
	// DefaultProbeTick is how often Ensure re-probes a starting daemon.
	DefaultProbeTick = 250 * time.Millisecond
	// SpawnLockStale is how old a spawn lock may get before it is treated
	// as left behind by a crashed spawner and taken over.
	SpawnLockStale = 60 * time.Second
	// LogTailLines is how much of daemon.log a start failure carries.
	LogTailLines = 20
)

// StateDir returns ~/.kongtrol.
func StateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kongtrol")
}

// LogPath returns ~/.kongtrol/daemon.log, where the detached daemon's
// stdout/stderr go.
func LogPath() string { return filepath.Join(StateDir(), "daemon.log") }

// LockPath returns ~/.kongtrol/run/spawn.lock, which guards the
// probe-then-spawn window so two viewers racing each other (two `up`, or
// `up` and the tray) don't both start a daemon.
func LockPath() string { return filepath.Join(StateDir(), "run", "spawn.lock") }

// StartError wraps a daemon that died or never answered during startup;
// LogTail holds the last lines it wrote.
type StartError struct {
	Cause   error
	LogTail []string
}

func (e *StartError) Error() string { return e.Cause.Error() }
func (e *StartError) Unwrap() error { return e.Cause }

// Spawner makes sure a daemon answers, starting a detached one if needed.
type Spawner struct {
	// Probe reports whether the daemon API answers. Required.
	Probe func() bool
	// Spawn starts the daemon with output appended to logPath and returns
	// a channel that receives its exit result. Required; see Spawn.
	Spawn func(logPath string) (<-chan error, error)
	// LogPath and LockPath default to LogPath() and LockPath().
	LogPath, LockPath string
	// StartTimeout and ProbeTick default to DefaultStartTimeout and
	// DefaultProbeTick.
	StartTimeout, ProbeTick time.Duration
	// Lang selects the language of returned errors.
	Lang i18n.Lang
	// Base (the daemon's API base URL, optional) lets Ensure explain a port
	// held by another user's program (*ForeignListenerError, ADR-0062)
	// instead of starting a daemon that cannot bind it.
	Base string
}

// Ensure makes sure a daemon answers, spawning a detached one when none
// does. started reports whether this call spawned it. A daemon that dies
// or never answers yields a *StartError; a port another user holds yields
// a *ForeignListenerError.
func (s *Spawner) Ensure(ctx context.Context) (started bool, err error) {
	if s.Probe() {
		return false, nil
	}
	if err := s.checkListener(); err != nil {
		return false, err
	}
	logPath, lockPath := s.LogPath, s.LockPath
	if logPath == "" {
		logPath = LogPath()
	}
	if lockPath == "" {
		lockPath = LockPath()
	}

	release, owner := AcquireSpawnLock(lockPath)
	if !owner {
		// Another viewer is spawning right now — wait for its daemon
		// instead of starting a second one.
		if s.waitForAPI(ctx, nil) == nil {
			return false, nil
		}
		return false, fmt.Errorf("%s", i18n.T(s.Lang, "cli.daemon.up.other_spawn_timeout"))
	}
	defer release()

	// Re-check under the lock: the racing viewer may have finished between
	// the first probe and taking the lock.
	if s.Probe() {
		return false, nil
	}

	var offset int64
	if fi, statErr := os.Stat(logPath); statErr == nil {
		offset = fi.Size()
	}
	exited, err := s.Spawn(logPath)
	if err != nil {
		return false, fmt.Errorf("%s", i18n.F(s.Lang, "cli.daemon.up.spawn_failed", err))
	}
	if err := s.waitForAPI(ctx, exited); err != nil {
		if errors.Is(err, context.Canceled) {
			return false, err
		}
		return false, &StartError{Cause: err, LogTail: ReadLogTail(logPath, offset, LogTailLines)}
	}
	return true, nil
}

func (s *Spawner) checkListener() error {
	if s.Base == "" {
		return nil
	}
	var fe *ForeignListenerError
	if errors.As(checkListenerFn(s.Base), &fe) {
		return &ForeignListenerError{Addr: fe.Addr, Lang: s.Lang}
	}
	return nil
}

// checkListenerFn is CheckListener (a test seam).
var checkListenerFn = CheckListener

// waitForAPI polls until the daemon answers, the spawned process exits
// (exited, nil when not ours), ctx ends or the start timeout passes.
func (s *Spawner) waitForAPI(ctx context.Context, exited <-chan error) error {
	timeout, tickEvery := s.StartTimeout, s.ProbeTick
	if timeout <= 0 {
		timeout = DefaultStartTimeout
	}
	if tickEvery <= 0 {
		tickEvery = DefaultProbeTick
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(tickEvery)
	defer tick.Stop()
	for {
		if s.Probe() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-exited:
			if err == nil {
				err = errors.New("exit status 0")
			}
			return fmt.Errorf("%s", i18n.F(s.Lang, "cli.daemon.up.died", err))
		case <-deadline.C:
			return fmt.Errorf("%s", i18n.F(s.Lang, "cli.daemon.up.start_timeout", timeout))
		case <-tick.C:
		}
	}
}

// AcquireSpawnLock creates the spawn lock exclusively. owner is false when
// another live spawner holds it; a lock older than SpawnLockStale is
// assumed abandoned and taken over.
func AcquireSpawnLock(path string) (release func(), owner bool) {
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(path) }, true
		}
		if !os.IsExist(err) {
			// Can't lock at all (permissions, odd FS): proceed unlocked
			// rather than block startup entirely.
			return func() {}, true
		}
		fi, statErr := os.Stat(path)
		if statErr != nil || time.Since(fi.ModTime()) < SpawnLockStale {
			return func() {}, false
		}
		_ = os.Remove(path)
	}
	return func() {}, false
}

// Spawn starts exe with args fully detached from the caller (no console
// window, own session/process group — see sysProcAttr per OS), with
// stdout and stderr appended to logPath and the home directory as working
// directory. The returned channel receives the process's exit result.
func Spawn(exe string, args []string, logPath string, lang i18n.Lang) (<-chan error, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	// The child holds its own handle; ours is only needed until Start.
	defer logFile.Close()
	_, _ = fmt.Fprintln(logFile, i18n.F(lang, "cli.daemon.log.spawned", time.Now().Format(time.RFC3339), os.Getpid()))

	newCmd := func(breakaway bool) *exec.Cmd {
		c := exec.Command(exe, args...)
		c.Stdout = logFile
		c.Stderr = logFile
		if home, err := os.UserHomeDir(); err == nil {
			// Don't pin the caller's current directory for the daemon's
			// lifetime (on Windows that blocks deleting/renaming it).
			c.Dir = home
		}
		c.SysProcAttr = sysProcAttr(breakaway)
		return c
	}
	cmd := newCmd(true)
	if err := cmd.Start(); err != nil {
		// The breakaway-from-job request is refused inside jobs that
		// forbid it; retry with plain detachment.
		cmd = newCmd(false)
		if err := cmd.Start(); err != nil {
			return nil, err
		}
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}

// FindKongtrol locates the kongtrol CLI for a companion binary (the tray):
// next to the running executable first, so an installed pair always uses
// its own CLI, then PATH. The error is translated.
func FindKongtrol(lang i18n.Lang) (string, error) {
	return findCompanion(executableName, lang, "daemonctl.kongtrol_not_found")
}

// FindTray locates the installed desktop companion next to the CLI or on PATH.
func FindTray(lang i18n.Lang) (string, error) {
	name := "kongtrol-tray" + filepath.Ext(executableName)
	return findCompanion(name, lang, "daemonctl.tray_not_found")
}

func findCompanion(name string, lang i18n.Lang, missingKey string) (string, error) {
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil {
			self = resolved
		}
		candidate := filepath.Join(filepath.Dir(self), name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs, nil
		}
		return p, nil
	}
	return "", fmt.Errorf("%s", i18n.F(lang, missingKey, name))
}

// ReadLogTail returns up to n trailing non-empty lines written to path at
// or after offset.
func ReadLogTail(path string, offset int64, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil
		}
	}
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines
}
