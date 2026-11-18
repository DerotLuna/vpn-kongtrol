package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// auditState is the running audit logger and the settings it was opened
// with. Swapped atomically: logAudit runs on many goroutines while a
// settings change from the dashboard reopens the logger.
type auditState struct {
	logger *security.AuditLogger
	conf   config.AuditLogConfig
}

var (
	auditCur atomic.Pointer[auditState]
	auditMu  sync.Mutex // serializes reopen/close
)

// currentAudit returns the running audit logger (nil when off).
func currentAudit() *security.AuditLogger {
	if st := auditCur.Load(); st != nil {
		return st.logger
	}
	return nil
}

// applyAuditConfig opens, reopens or closes the audit logger so it matches
// c.Security.AuditLog — at boot and whenever the settings change (path,
// signing, size per file, retention), without a daemon restart.
func applyAuditConfig(c *config.Config) {
	if c == nil {
		return
	}
	want := c.Security.AuditLog
	auditMu.Lock()
	defer auditMu.Unlock()
	if st := auditCur.Load(); st != nil && st.conf == want {
		return
	}
	var next *auditState
	if want.Path != "" {
		var key []byte
		if want.Sign {
			key = loadOrCreateAuditHMACKey()
		}
		a, err := security.NewAuditLogger(want.Path, security.AuditLoggerOptions{
			Sign:          want.Sign,
			Key:           key,
			MaxSizeBytes:  int64(want.MaxSizeMB) << 20,
			RetentionDays: want.RetentionDays,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.audit.init_warn", err)))
		} else {
			next = &auditState{logger: a, conf: want}
		}
	}
	if old := auditCur.Swap(next); old != nil {
		_ = old.logger.Close()
	}
}

// closeAudit stops writing the audit log (purge, shutdown).
func closeAudit() {
	auditMu.Lock()
	defer auditMu.Unlock()
	if old := auditCur.Swap(nil); old != nil {
		_ = old.logger.Close()
	}
}

// auditVerifier checks an entry's signature with the key in the OS
// keychain. It reads an existing key only (never creates one), so entries
// signed before signing was turned off still verify. The key never leaves
// the daemon: clients only get the true/false answer.
func auditVerifier() func(security.AuditEvent) bool {
	var (
		mu  sync.Mutex
		key []byte
	)
	return func(ev security.AuditEvent) bool {
		if ev.HMAC == "" {
			return false
		}
		mu.Lock()
		if key == nil {
			key = readAuditHMACKey()
		}
		k := key
		mu.Unlock()
		if len(k) == 0 {
			return false
		}
		return security.VerifyEvent(ev, k)
	}
}

// readAuditHMACKey returns the stored signing key, or nil when none.
func readAuditHMACKey() []byte {
	s, err := config.GetCredential(auditHMACKeychainProfile, auditHMACKeychainKey)
	if err != nil || s == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}
