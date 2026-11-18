package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

var (
	logsFollow  bool
	logsTail    int
	logsProfile string
	logsLevel   string
	logsSince   string
	logsUntil   string
)

var logsCmd = &cobra.Command{
	Use: "logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil || cfg.Security.AuditLog.Path == "" {
			return fmt.Errorf("%s", ct("cli.logs.audit_not_configured"))
		}
		q, err := logsQuery()
		if err != nil {
			return err
		}
		// Newest first from the reader; print oldest first like a log.
		events, _, err := security.QueryAuditEvents(cfg.Security.AuditLog.Path, q)
		if err != nil {
			return err
		}
		slices.Reverse(events)
		verify := auditVerifier()
		entries := make([]auditEntry, 0, len(events))
		for _, ev := range events {
			entries = append(entries, auditEntry{AuditEvent: ev, Signed: ev.HMAC != "", Verified: verify(ev)})
		}
		if outputJSON {
			if err := emitJSON(entries); err != nil {
				return err
			}
		} else {
			for _, e := range entries {
				printAuditEvent(e.AuditEvent)
				if e.Signed && !e.Verified {
					fmt.Println("    " + tuiWarn(ct("cli.logs.bad_signature")))
				}
			}
		}
		if !logsFollow {
			return nil
		}
		seenByPath, err := auditSeenByPath(cfg.Security.AuditLog.Path)
		if err != nil {
			return err
		}
		return followAuditLog(cfg.Security.AuditLog.Path, seenByPath)
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, ct("cli.logs.flag.follow"))
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, ct("cli.logs.flag.tail"))
	logsCmd.Flags().StringVar(&logsProfile, "profile", "", ct("cli.logs.flag.profile"))
	logsCmd.Flags().StringVar(&logsLevel, "level", "", ct("cli.logs.flag.level"))
	logsCmd.Flags().StringVar(&logsSince, "since", "", ct("cli.logs.flag.since"))
	logsCmd.Flags().StringVar(&logsUntil, "until", "", ct("cli.logs.flag.until"))
	rootCmd.AddCommand(logsCmd)
}

// auditEntry is one `logs --json` entry, the same shape as GET /api/v1/audit:
// the event plus whether it is signed and its signature checks out.
type auditEntry struct {
	security.AuditEvent
	Signed   bool `json:"signed"`
	Verified bool `json:"verified"`
}

// logsQuery builds the reader query from the flags: --tail newest events
// (0 = as many as one page allows), --since/--until as local days
// (YYYY-MM-DD, --until inclusive).
func logsQuery() (security.AuditQuery, error) {
	q := security.AuditQuery{Profile: logsProfile, Level: logsLevel, Limit: logsTail}
	if q.Limit <= 0 || q.Limit > security.MaxAuditQueryLimit {
		q.Limit = security.MaxAuditQueryLimit
	}
	for _, f := range []struct {
		raw string
		dst *time.Time
		end bool
	}{{logsSince, &q.From, false}, {logsUntil, &q.To, true}} {
		if f.raw == "" {
			continue
		}
		d, err := time.ParseInLocation("2006-01-02", f.raw, time.Local)
		if err != nil {
			return q, fmt.Errorf("%s", cf("cli.logs.bad_date", f.raw))
		}
		if f.end {
			d = d.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		*f.dst = d
	}
	return q, nil
}

// dailyLogPaths returns the audit files that are candidates for "current"
// activity: yesterday's (in case midnight just rolled over and a writer
// still has it open) and today's, every size segment included, plus
// today's day file even before it exists.
func dailyLogPaths(basePath string) []string {
	now := time.Now()
	paths := security.AuditDayFiles(basePath, now.AddDate(0, 0, -1))
	paths = append(paths, security.AuditDayFiles(basePath, now)...)
	if today := security.DailyAuditLogPath(basePath, now); !slices.Contains(paths, today) {
		paths = append(paths, today)
	}
	return paths
}

func filterAuditEvents(in []security.AuditEvent) []security.AuditEvent {
	out := make([]security.AuditEvent, 0, len(in))
	level := strings.ToUpper(strings.TrimSpace(logsLevel))
	for _, ev := range in {
		if logsProfile != "" && ev.Profile != logsProfile {
			continue
		}
		if level != "" && strings.ToUpper(ev.Level) != level {
			continue
		}
		out = append(out, ev)
	}
	return out
}

func printAuditEvent(ev security.AuditEvent) {
	t := ev.Timestamp.Local().Format("15:04:05")
	badge := renderLevelBadge(ev.Level)

	profile := ev.Profile
	if profile == "" {
		profile = "—"
	}
	fmt.Printf("%s  %s  %s  %s  %s\n",
		styleDim.Render(t),
		badge,
		styleStatusName.Render(fmt.Sprintf("%-20s", ev.Action)),
		styleStatusIP.Render(fmt.Sprintf("%-14s", profile)),
		styleMuted.Render(ev.Message))
}

func renderLevelBadge(level string) string {
	level = strings.ToUpper(strings.TrimSpace(level))
	switch level {
	case "ERROR":
		return styleErr.Render(fmt.Sprintf("%s %-8s", sym("✗", "x"), level))
	case "WARN":
		return styleWarn.Render(fmt.Sprintf("%s %-8s", sym("⚠", "!"), level))
	case "SECURITY":
		return styleGold.Render(fmt.Sprintf("%s %-8s", "●", level))
	case "OK":
		return styleOK.Render(fmt.Sprintf("%s %-8s", sym("✓", "ok"), level))
	default:
		if level == "" {
			level = "INFO"
		}
		return styleDim.Render(fmt.Sprintf("%s %-8s", sym("·", "-"), level))
	}
}

// auditSeenByPath returns the current byte size of each candidate daily
// log file, used as the starting tail offset — everything up to that
// offset was already printed by the initial (non-follow) read.
func auditSeenByPath(basePath string) (map[string]int64, error) {
	offsets := map[string]int64{}
	for _, path := range dailyLogPaths(basePath) {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				offsets[path] = 0
				continue
			}
			return nil, fmt.Errorf("logs: stat: %w", err)
		}
		offsets[path] = info.Size()
	}
	return offsets, nil
}

// auditTailer incrementally reads newly appended audit events from a set
// of daily log files, tracking a byte offset per path so each poll only
// reads the bytes written since the last poll instead of re-parsing the
// whole file.
type auditTailer struct {
	offsets map[string]int64
	partial map[string][]byte // trailing bytes of an incomplete last line, per path
}

func newAuditTailer(offsets map[string]int64) *auditTailer {
	if offsets == nil {
		offsets = map[string]int64{}
	}
	return &auditTailer{offsets: offsets, partial: map[string][]byte{}}
}

func (t *auditTailer) poll(path string) ([]security.AuditEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("logs: open: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("logs: stat: %w", err)
	}

	offset := t.offsets[path]
	if info.Size() < offset {
		// File shrank (truncated or replaced under the same name) — restart.
		offset = 0
		t.partial[path] = nil
	}
	if info.Size() == offset {
		return nil, nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("logs: seek: %w", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("logs: read: %w", err)
	}
	t.offsets[path] = offset + int64(len(data))

	buf := append(t.partial[path], data...)
	lines := bytes.Split(buf, []byte("\n"))
	t.partial[path] = append([]byte(nil), lines[len(lines)-1]...)
	lines = lines[:len(lines)-1]

	var out []security.AuditEvent
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev security.AuditEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

func (t *auditTailer) prune(active map[string]bool) {
	for p := range t.offsets {
		if !active[p] {
			delete(t.offsets, p)
			delete(t.partial, p)
		}
	}
}

// followAuditLog streams new audit events as they're written, waking on
// filesystem change notifications (inotify / FSEvents / ReadDirectoryChangesW,
// depending on OS) instead of polling the whole file on a timer. A slow
// fallback ticker covers watchers that can miss events (network mounts,
// write bursts) — it's cheap now since poll() only ever reads from the
// last offset, not the whole file.
func followAuditLog(basePath string, offsets map[string]int64) error {
	tailer := newAuditTailer(offsets)
	dir := filepath.Dir(basePath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("logs: watcher: %w", err)
	}
	defer watcher.Close()
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("logs: watch %s: %w", dir, err)
	}

	emit := func(path string) error {
		events, err := tailer.poll(path)
		if err != nil {
			return err
		}
		for _, ev := range filterAuditEvents(events) {
			if outputJSON {
				b, _ := json.Marshal(ev)
				_, _ = io.WriteString(os.Stdout, string(b)+"\n")
			} else {
				printAuditEvent(ev)
			}
		}
		return nil
	}

	checkActivePaths := func() error {
		paths := dailyLogPaths(basePath)
		active := make(map[string]bool, len(paths))
		for _, p := range paths {
			active[p] = true
		}
		for _, p := range paths {
			if err := emit(p); err != nil {
				return err
			}
		}
		tailer.prune(active)
		return nil
	}

	fallback := time.NewTicker(5 * time.Second)
	defer fallback.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if err := checkActivePaths(); err != nil {
				return err
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("logs: watch: %w", err)
		case <-fallback.C:
			if err := checkActivePaths(); err != nil {
				return err
			}
		}
	}
}
