// Package security provides kill switch, DNS guard, leak detection, and audit logging.
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditEvent represents a single tamper-evident audit log entry.
type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`   // INFO | WARN | ERROR | SECURITY
	Action    string    `json:"action"`  // vpn.connect | vpn.disconnect | security.killswitch | etc.
	Profile   string    `json:"profile"` // VPN profile name, if applicable
	Message   string    `json:"message"`
	HMAC      string    `json:"hmac"` // hex-encoded HMAC-SHA256 of the rest of the fields
}

const (
	// DefaultAuditRetentionDays is the number of days kept when
	// retention_days is 0 or unset (ADR-0064; before it, 0 kept only today
	// and yesterday).
	DefaultAuditRetentionDays = 30
	// MaxAuditRetentionDays bounds security.audit_log.retention_days.
	MaxAuditRetentionDays = 365
	// MaxAuditFileSizeMB bounds security.audit_log.max_size_mb.
	MaxAuditFileSizeMB = 1024
	// DefaultAuditMaxTotalMB is the space all audit files may use together
	// when max_total_mb is 0 or unset (never below one file's size).
	DefaultAuditMaxTotalMB = 500
	// MaxAuditTotalMB bounds security.audit_log.max_total_mb.
	MaxAuditTotalMB = 10240
)

// AuditLoggerOptions configures an AuditLogger.
type AuditLoggerOptions struct {
	// Sign adds an HMAC-SHA256 to every entry (needs Key).
	Sign bool
	Key  []byte
	// MaxSizeBytes is the largest size of one audit file: when the next
	// entry would pass it, the logger continues the same day in
	// "<base>-YYYY-MM-DD.1<ext>", ".2", … (like logrotate's maxsize).
	// 0 = no size limit (one file per day).
	MaxSizeBytes int64
	// MaxTotalBytes caps the space of all audit files together: the
	// oldest files are deleted first, keeping room for the file being
	// written to reach MaxSizeBytes. 0 = no total cap.
	MaxTotalBytes int64
	// RetentionDays keeps the files of the last N local days (today
	// included); older days are deleted. 0 = DefaultAuditRetentionDays.
	RetentionDays int
	// Now and After replace the clock in tests (nil = time.Now and
	// time.After).
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time
}

// AuditLogger writes signed, append-only audit events to daily log files.
// Old files are deleted when it opens, when a file fills up and every
// local midnight, so the log never keeps more than RetentionDays days or
// MaxTotalBytes in total.
type AuditLogger struct {
	mu        sync.Mutex
	basePath  string
	dateKey   string
	seg       int
	size      int64
	f         *os.File
	sign      bool
	hmacKey   []byte
	maxSize   int64
	maxTotal  int64
	retention int
	now       func() time.Time
	after     func(time.Duration) <-chan time.Time
	stop      chan struct{}
	stopOnce  sync.Once
	done      chan struct{}
}

// NewAuditLogger creates or opens today's audit log for the base path
// (the logger never writes path itself, only its daily files) and starts
// the daily cleanup; Close stops it.
func NewAuditLogger(path string, opts AuditLoggerOptions) (*AuditLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("audit: create log dir: %w", err)
	}
	l := &AuditLogger{
		basePath:  path,
		sign:      opts.Sign,
		hmacKey:   opts.Key,
		maxSize:   opts.MaxSizeBytes,
		maxTotal:  opts.MaxTotalBytes,
		retention: EffectiveAuditRetentionDays(opts.RetentionDays),
		now:       opts.Now,
		after:     opts.After,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	if l.now == nil {
		l.now = time.Now
	}
	if l.after == nil {
		l.after = time.After
	}
	if err := l.openDay(l.now()); err != nil {
		return nil, err
	}
	go l.dailyCleanup()
	return l, nil
}

// EffectiveAuditRetentionDays is the number of days really kept for a
// configured retention_days: 0 (unset) = DefaultAuditRetentionDays,
// capped at MaxAuditRetentionDays.
func EffectiveAuditRetentionDays(days int) int {
	switch {
	case days <= 0:
		return DefaultAuditRetentionDays
	case days > MaxAuditRetentionDays:
		return MaxAuditRetentionDays
	}
	return days
}

// untilNextMidnight is the wait from now to the next local midnight, plus
// one second so the new day has surely begun.
func untilNextMidnight(now time.Time) time.Duration {
	local := now.In(time.Local)
	next := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 1, 0, time.Local)
	return next.Sub(local)
}

// dailyCleanup deletes old files every local midnight, also on days
// nothing is logged (otherwise only the next entry would clean up).
func (l *AuditLogger) dailyCleanup() {
	defer close(l.done)
	for {
		select {
		case <-l.stop:
			return
		case <-l.after(untilNextMidnight(l.now())):
		}
		l.mu.Lock()
		now := l.now()
		// The day changed: close yesterday's file (the next entry opens
		// today's), so retention_days: 1 can delete it.
		if l.f != nil && now.In(time.Local).Format(auditDateLayout) != l.dateKey {
			_ = l.f.Close()
			l.f = nil
		}
		l.pruneLocked(now)
		l.mu.Unlock()
	}
}

// Log writes a new audit event.
func (l *AuditLogger) Log(level, action, profile, message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	ev := AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: now.UTC(),
		Level:     level,
		Action:    action,
		Profile:   profile,
		Message:   message,
	}
	if l.sign && len(l.hmacKey) > 0 {
		ev.HMAC = computeHMAC(ev, l.hmacKey)
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("audit: marshal event: %w", err)
	}
	line = append(line, '\n')

	if err := l.rotateIfNeeded(now, int64(len(line))); err != nil {
		return err
	}
	n, err := l.f.Write(line)
	l.size += int64(n)
	return err
}

// Verify reports whether ev carries a valid signature from this logger's
// key: false for an unsigned entry, a tampered one, or no key.
func (l *AuditLogger) Verify(ev AuditEvent) bool {
	if l == nil || ev.HMAC == "" || len(l.hmacKey) == 0 {
		return false
	}
	return VerifyEvent(ev, l.hmacKey)
}

// Info logs an informational audit event.
func (l *AuditLogger) Info(action, profile, message string) {
	_ = l.Log("INFO", action, profile, message)
}

// Security logs a security-relevant audit event.
func (l *AuditLogger) Security(action, profile, message string) {
	_ = l.Log("SECURITY", action, profile, message)
}

// Warn logs a warning audit event.
func (l *AuditLogger) Warn(action, profile, message string) {
	_ = l.Log("WARN", action, profile, message)
}

// Close stops the daily cleanup and closes the log file.
func (l *AuditLogger) Close() error {
	if l.stop != nil {
		l.stopOnce.Do(func() {
			close(l.stop)
			<-l.done
		})
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// openDay opens the newest segment of now's day (creating the day file).
func (l *AuditLogger) openDay(now time.Time) error {
	key := now.In(time.Local).Format(auditDateLayout)
	seg := 0
	if files := auditDayFiles(l.basePath, key); len(files) > 0 {
		seg = files[len(files)-1].Seg
	}
	return l.openSegment(now, key, seg)
}

// openSegment opens one file, then deletes what the retention and the
// total cap no longer allow.
func (l *AuditLogger) openSegment(now time.Time, dateKey string, seg int) error {
	if l.f != nil {
		_ = l.f.Close()
		l.f = nil
	}
	path := auditSegmentPath(l.basePath, dateKey, seg)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: open log %s: %w", path, err)
	}
	var size int64
	if fi, err := f.Stat(); err == nil {
		size = fi.Size()
	}
	l.f, l.dateKey, l.seg, l.size = f, dateKey, seg, size
	l.pruneLocked(now)
	return nil
}

// rotateIfNeeded moves to a new day's file at midnight, and to the next
// segment of the day when the next line would pass MaxSizeBytes. A file
// always takes at least one line, so an oversized entry is never lost.
func (l *AuditLogger) rotateIfNeeded(now time.Time, next int64) error {
	if l.basePath == "" {
		return nil
	}
	key := now.In(time.Local).Format(auditDateLayout)
	if l.f == nil || key != l.dateKey {
		return l.openDay(now)
	}
	if l.maxSize > 0 && l.size > 0 && l.size+next > l.maxSize {
		return l.openSegment(now, key, l.seg+1)
	}
	return nil
}

// pruneLocked deletes the files of days older than the retention, then the
// oldest files until all of them fit in MaxTotalBytes with room for the
// open file to reach MaxSizeBytes. The open file is never deleted.
func (l *AuditLogger) pruneLocked(now time.Time) {
	if l.basePath == "" {
		return
	}
	current := ""
	if l.f != nil {
		current = filepath.Clean(auditSegmentPath(l.basePath, l.dateKey, l.seg))
	}
	oldest := now.In(time.Local).AddDate(0, 0, -(EffectiveAuditRetentionDays(l.retention) - 1)).Format(auditDateLayout)
	type sized struct {
		path string
		size int64
	}
	var others []sized
	var total int64
	for _, af := range listAuditFiles(l.basePath) {
		if filepath.Clean(af.Path) == current {
			continue
		}
		if af.Date < oldest {
			_ = os.Remove(af.Path)
			continue
		}
		fi, err := os.Stat(af.Path)
		if err != nil {
			continue
		}
		others = append(others, sized{af.Path, fi.Size()})
		total += fi.Size()
	}
	if l.maxTotal <= 0 {
		return
	}
	var reserve int64
	if l.f != nil {
		reserve = max(l.size, l.maxSize)
	}
	// listAuditFiles is oldest first, so the oldest go first.
	for i := 0; i < len(others) && total+reserve > l.maxTotal; i++ {
		if err := os.Remove(others[i].path); err == nil {
			total -= others[i].size
		}
	}
}

const auditDateLayout = "2006-01-02"

// auditFile is one file the logger wrote: day file (Seg 0) or a size
// segment of that day (Seg ≥ 1).
type auditFile struct {
	Path string
	Date string // YYYY-MM-DD, local
	Seg  int
}

// DailyAuditLogPath returns the daily rotated path for a base log path.
// Example: /var/log/audit.log -> /var/log/audit-2026-07-15.log
func DailyAuditLogPath(basePath string, at time.Time) string {
	return auditSegmentPath(basePath, at.In(time.Local).Format(auditDateLayout), 0)
}

// auditSegmentPath: seg 0 is the day file; seg N is "<base>-DATE.N<ext>".
func auditSegmentPath(basePath, dateKey string, seg int) string {
	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext) + "-" + dateKey
	if seg > 0 {
		name += "." + strconv.Itoa(seg)
	}
	return name + ext
}

// listAuditFiles lists the files the logger wrote for basePath, oldest
// day first and segments in order.
func listAuditFiles(basePath string) []auditFile {
	if basePath == "" {
		return nil
	}
	// Glob returns OS separators; compare against the same form.
	basePath = filepath.Clean(basePath)
	ext := filepath.Ext(basePath)
	baseNoExt := strings.TrimSuffix(basePath, ext)
	paths, err := filepath.Glob(globEscape(baseNoExt) + "-*" + globEscape(ext))
	if err != nil {
		return nil
	}
	var out []auditFile
	for _, p := range paths {
		if af, ok := parseAuditFile(basePath, p); ok {
			out = append(out, af)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].Seg < out[j].Seg
	})
	return out
}

// globEscape quotes the glob metacharacters of a literal path part.
func globEscape(s string) string {
	r := strings.NewReplacer(`[`, `[[]`, `*`, `[*]`, `?`, `[?]`)
	return r.Replace(s)
}

// auditDayFiles lists the segments of one day, in write order.
func auditDayFiles(basePath, dateKey string) []auditFile {
	var out []auditFile
	for _, af := range listAuditFiles(basePath) {
		if af.Date == dateKey {
			out = append(out, af)
		}
	}
	return out
}

// AuditDayFiles returns the files of the local day of at, in write order
// (the day file first, then its size segments). Missing days give nil.
func AuditDayFiles(basePath string, at time.Time) []string {
	var out []string
	for _, af := range auditDayFiles(basePath, at.In(time.Local).Format(auditDateLayout)) {
		out = append(out, af.Path)
	}
	return out
}

// AuditLogFiles lists the files the audit logger actually wrote for
// basePath: only "<base>-YYYY-MM-DD[.N]<ext>" names (the logger never
// writes basePath itself). Used by purge, so a user-set path never makes it
// delete anything but these files.
func AuditLogFiles(basePath string) []string {
	var out []string
	for _, af := range listAuditFiles(basePath) {
		out = append(out, af.Path)
	}
	return out
}

func parseAuditFile(basePath, path string) (auditFile, bool) {
	ext := filepath.Ext(basePath)
	baseNoExt := strings.TrimSuffix(basePath, ext)
	if !strings.HasPrefix(path, baseNoExt+"-") || !strings.HasSuffix(path, ext) {
		return auditFile{}, false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(path, baseNoExt+"-"), ext)
	if len(rest) < len(auditDateLayout) {
		return auditFile{}, false
	}
	date, tail := rest[:len(auditDateLayout)], rest[len(auditDateLayout):]
	if _, err := time.Parse(auditDateLayout, date); err != nil {
		return auditFile{}, false
	}
	seg := 0
	if tail != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(tail, "."))
		// ".N" only: N ≥ 1, no sign, no leading zero.
		if !strings.HasPrefix(tail, ".") || err != nil || n < 1 || strconv.Itoa(n) != tail[1:] {
			return auditFile{}, false
		}
		seg = n
	}
	return auditFile{Path: path, Date: date, Seg: seg}, true
}

// computeHMAC computes the HMAC-SHA256 of the event's content fields (excluding HMAC itself).
func computeHMAC(ev AuditEvent, key []byte) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		ev.ID, ev.Timestamp.Format(time.RFC3339Nano),
		ev.Level, ev.Action, ev.Profile, ev.Message)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyEvent checks the HMAC of a log entry.
// Returns true if the entry is authentic, false if it has been tampered with.
// An unsigned entry (signing was off) returns true: callers that must tell
// "signed and valid" apart use AuditLogger.Verify or check ev.HMAC.
func VerifyEvent(ev AuditEvent, key []byte) bool {
	if ev.HMAC == "" {
		return true // signing was disabled
	}
	expected := computeHMAC(ev, key)
	return hmac.Equal([]byte(ev.HMAC), []byte(expected))
}
