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
	mu      sync.Mutex
	path    string
	f       *os.File
	sign    bool
	hmacKey []byte
}

// NewAuditLogger creates or opens the audit log at path.
// Pass hmacKey=nil to disable signing (sign=false in config).
func NewAuditLogger(path string, sign bool, hmacKey []byte) (*AuditLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("audit: create log dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit: open log %s: %w", path, err)
	}

	return &AuditLogger{
		path:    path,
		f:       f,
		sign:    sign,
		hmacKey: hmacKey,
	}, nil
}

// Log writes a new audit event.
func (l *AuditLogger) Log(level, action, profile, message string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

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
	return l.f.Close()
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
