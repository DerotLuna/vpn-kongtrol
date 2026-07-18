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

// AuditLogger writes signed, append-only audit events to a log file.
// The HMAC key must be passed at construction — never stored in the struct
// after initialization to avoid keeping it in memory longer than necessary.
type AuditLogger struct {
	mu       sync.Mutex
	basePath string
	dateKey  string
	f        *os.File
	sign     bool
	hmacKey  []byte
}

// NewAuditLogger creates or opens the audit log at path.
// Pass hmacKey=nil to disable signing (sign=false in config).
func NewAuditLogger(path string, sign bool, hmacKey []byte) (*AuditLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("audit: create log dir: %w", err)
	}
	now := time.Now()
	dailyPath := DailyAuditLogPath(path, now)
	f, err := os.OpenFile(dailyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit: open log %s: %w", dailyPath, err)
	}

	l := &AuditLogger{
		basePath: path,
		f:        f,
		dateKey:  now.In(time.Local).Format("2006-01-02"),
		sign:     sign,
		hmacKey:  hmacKey,
	}
	l.pruneOldFiles(now)
	return l, nil
}

// Log writes a new audit event.
func (l *AuditLogger) Log(level, action, profile, message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.rotateIfNeeded(time.Now()); err != nil {
		return err
	}

	ev := AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
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

	_, err = fmt.Fprintf(l.f, "%s\n", line)
	return err
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

// Close flushes and closes the log file.
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	return l.f.Close()
}

func (l *AuditLogger) rotateIfNeeded(now time.Time) error {
	if l.basePath == "" {
		return nil
	}
	targetKey := now.In(time.Local).Format("2006-01-02")
	if l.f != nil && targetKey == l.dateKey {
		return nil
	}
	if l.f != nil {
		_ = l.f.Close()
	}
	path := DailyAuditLogPath(l.basePath, now)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: open log %s: %w", path, err)
	}
	l.f = f
	l.dateKey = targetKey
	l.pruneOldFiles(now)
	return nil
}

func (l *AuditLogger) pruneOldFiles(now time.Time) {
	if l.basePath == "" {
		return
	}
	todayKey := now.In(time.Local).Format("2006-01-02")
	yesterdayKey := now.In(time.Local).AddDate(0, 0, -1).Format("2006-01-02")
	for _, p := range DailyAuditLogCandidates(l.basePath, 32) {
		dk, ok := parseDailyAuditDateKey(l.basePath, p)
		if !ok || dk == todayKey || dk == yesterdayKey {
			continue
		}
		_ = os.Remove(p)
	}
}

// DailyAuditLogPath returns the daily rotated path for a base log path.
// Example: /var/log/audit.log -> /var/log/audit-2026-07-15.log
func DailyAuditLogPath(basePath string, at time.Time) string {
	date := at.In(time.Local).Format("2006-01-02")
	ext := filepath.Ext(basePath)
	if ext == "" {
		return basePath + "-" + date
	}
	baseNoExt := strings.TrimSuffix(basePath, ext)
	return baseNoExt + "-" + date + ext
}

// DailyAuditLogCandidates returns rotated audit files matching the base path
// suffix pattern, newest first, capped by limit.
func DailyAuditLogCandidates(basePath string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	ext := filepath.Ext(basePath)
	baseNoExt := strings.TrimSuffix(basePath, ext)
	pattern := baseNoExt + "-*" + ext
	paths, err := filepath.Glob(pattern)
	if err != nil || len(paths) == 0 {
		return nil
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i] > paths[j] })
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return paths
}

func parseDailyAuditDateKey(basePath, path string) (string, bool) {
	ext := filepath.Ext(basePath)
	baseNoExt := strings.TrimSuffix(basePath, ext)
	if !strings.HasPrefix(path, baseNoExt+"-") || !strings.HasSuffix(path, ext) {
		return "", false
	}
	datePart := strings.TrimPrefix(path, baseNoExt+"-")
	datePart = strings.TrimSuffix(datePart, ext)
	if len(datePart) != len("2006-01-02") {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", datePart); err != nil {
		return "", false
	}
	return datePart, true
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
func VerifyEvent(ev AuditEvent, key []byte) bool {
	if ev.HMAC == "" {
		return true // signing was disabled
	}
	expected := computeHMAC(ev, key)
	return hmac.Equal([]byte(ev.HMAC), []byte(expected))
}
