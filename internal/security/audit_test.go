package security

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempAuditLog(t *testing.T, sign bool, key []byte) (*AuditLogger, string) {
	t.Helper()
	dir := t.TempDir()
	basePath := filepath.Join(dir, "audit.log")
	l, err := NewAuditLogger(basePath, sign, key)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l, DailyAuditLogPath(basePath, time.Now())
}

func readEvents(t *testing.T, path string) []AuditEvent {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()

	var events []AuditEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

func TestAuditLogger_WritesEvent(t *testing.T) {
	l, path := tempAuditLog(t, false, nil)

	if err := l.Log("INFO", "vpn.connect", "office", "connected successfully"); err != nil {
		t.Fatalf("Log: %v", err)
	}
	_ = l.Close()

	events := readEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", ev.Level)
	}
	if ev.Action != "vpn.connect" {
		t.Errorf("Action = %q, want vpn.connect", ev.Action)
	}
	if ev.Profile != "office" {
		t.Errorf("Profile = %q, want office", ev.Profile)
	}
	if ev.ID == "" {
		t.Error("ID should not be empty")
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAuditLogger_MultipleEvents(t *testing.T) {
	l, path := tempAuditLog(t, false, nil)

	l.Info("vpn.connect", "office", "msg1")
	l.Warn("vpn.drop", "office", "msg2")
	l.Security("security.killswitch", "", "msg3")
	_ = l.Close()

	events := readEvents(t, path)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestAuditLogger_SignedEntries_Valid(t *testing.T) {
	key := []byte("test-hmac-key-32-bytes-padded!!!")
	l, path := tempAuditLog(t, true, key)

	l.Info("vpn.connect", "server", "test signed entry")
	_ = l.Close()

	events := readEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.HMAC == "" {
		t.Fatal("HMAC should be set when signing is enabled")
	}
	if !VerifyEvent(ev, key) {
		t.Error("VerifyEvent returned false for a valid signed event")
	}
}

func TestAuditLogger_SignedEntries_TamperedFails(t *testing.T) {
	key := []byte("test-hmac-key-32-bytes-padded!!!")
	l, path := tempAuditLog(t, true, key)

	l.Info("vpn.connect", "server", "original message")
	_ = l.Close()

	events := readEvents(t, path)
	ev := events[0]

	// Tamper with the message.
	ev.Message = "tampered message"
	if VerifyEvent(ev, key) {
		t.Error("VerifyEvent should return false for a tampered event")
	}
}

func TestAuditLogger_NoSign_VerifyAlwaysTrue(t *testing.T) {
	l, path := tempAuditLog(t, false, nil)
	l.Info("test", "vpn", "no signing")
	_ = l.Close()

	events := readEvents(t, path)
	ev := events[0]

	if ev.HMAC != "" {
		t.Error("HMAC should be empty when signing is disabled")
	}
	// VerifyEvent with empty HMAC should return true (signing was disabled).
	if !VerifyEvent(ev, nil) {
		t.Error("VerifyEvent should return true when HMAC is empty (signing disabled)")
	}
}

func TestAuditLogger_UniqueIDs(t *testing.T) {
	l, path := tempAuditLog(t, false, nil)
	for i := 0; i < 10; i++ {
		l.Info("test", "vpn", "event")
	}
	_ = l.Close()

	events := readEvents(t, path)
	ids := make(map[string]bool)
	for _, ev := range events {
		if ids[ev.ID] {
			t.Errorf("duplicate event ID: %s", ev.ID)
		}
		ids[ev.ID] = true
	}
}

func TestAuditLogger_PrunesToTodayAndYesterday(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	older := DailyAuditLogPath(base, now.AddDate(0, 0, -2))
	yesterday := DailyAuditLogPath(base, now.AddDate(0, 0, -1))
	today := DailyAuditLogPath(base, now)
	for _, p := range []string{older, yesterday, today} {
		if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	l := &AuditLogger{basePath: base}
	l.pruneOldFiles(now)

	if _, err := os.Stat(today); err != nil {
		t.Fatalf("today should be kept: %v", err)
	}
	if _, err := os.Stat(yesterday); err != nil {
		t.Fatalf("yesterday should be kept: %v", err)
	}
	if _, err := os.Stat(older); !os.IsNotExist(err) {
		t.Fatalf("older file should be removed, err=%v", err)
	}
}
