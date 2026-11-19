package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// withTestAudit points logAudit at a fresh audit log for one test.
func withTestAudit(t *testing.T) func() []security.AuditEvent {
	t.Helper()
	base := filepath.Join(t.TempDir(), "audit.log")
	l, err := security.NewAuditLogger(base, security.AuditLoggerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	prev := auditCur.Swap(&auditState{logger: l})
	t.Cleanup(func() {
		auditCur.Store(prev)
		_ = l.Close()
	})
	return func() []security.AuditEvent {
		var out []security.AuditEvent
		for _, f := range security.AuditDayFiles(base, time.Now()) {
			evs, err := security.ReadAuditEvents(f)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, evs...)
		}
		return out
	}
}

func TestAuditAPIRequest_Rendered(t *testing.T) {
	read := withTestAudit(t)
	auditAPIRequest(api.AuditRecord{Level: "SECURITY", Action: "config.profile.update", Profile: "office", Client: "dashboard", Status: 200})
	auditAPIRequest(api.AuditRecord{Level: "WARN", Action: "config.policy.delete", Name: "work", Client: "app", Status: 404})
	auditAPIRequest(api.AuditRecord{Level: "SECURITY", Action: "auth.denied", Client: "unknown", Status: 401,
		Reason: "credential", Method: "PUT", Path: "/api/v1/vpns/office", Suppressed: 3})
	evs := read()
	if len(evs) != 3 {
		t.Fatalf("got %d entries, want 3", len(evs))
	}
	if evs[0].Action != "config.profile.update" || evs[0].Profile != "office" || evs[0].Level != "SECURITY" {
		t.Errorf("profile update entry: %+v", evs[0])
	}
	if !strings.Contains(evs[1].Message, "work") || !strings.Contains(evs[1].Message, "404") {
		t.Errorf("policy entry should name the policy and the result: %q", evs[1].Message)
	}
	if !strings.Contains(evs[2].Message, "/api/v1/vpns/office") || !strings.Contains(evs[2].Message, "3") {
		t.Errorf("denied entry: %q", evs[2].Message)
	}
	for _, ev := range evs {
		if strings.Contains(ev.Message, "cli.audit.") {
			t.Errorf("untranslated key in %q", ev.Message)
		}
	}
}

// The kill switch logs each state change once, not every re-apply.
func TestAuditKillSwitchState_OnlyChanges(t *testing.T) {
	read := withTestAudit(t)
	ksAuditMu.Lock()
	ksAuditState, ksAuditErr = "", ""
	ksAuditMu.Unlock()
	armed := security.KillSwitchStatus{State: security.KillSwitchArmed, Protected: []string{"office"}}
	active := security.KillSwitchStatus{State: security.KillSwitchActive, Blocking: []string{"office"}, BlockedCount: 3}
	auditKillSwitchState(armed)
	auditKillSwitchState(armed)
	auditKillSwitchState(active)
	auditKillSwitchState(active)
	failed := active
	failed.Error = "firewall refused"
	auditKillSwitchState(failed)
	evs := read()
	if len(evs) != 3 {
		t.Fatalf("got %d entries, want 3 (armed, active, error): %+v", len(evs), evs)
	}
	if evs[1].Level != "SECURITY" || evs[1].Profile != "office" || evs[1].Action != "security.killswitch.state" {
		t.Errorf("active entry: %+v", evs[1])
	}
	if evs[2].Action != "security.killswitch.error" || !strings.Contains(evs[2].Message, "firewall refused") {
		t.Errorf("error entry: %+v", evs[2])
	}
}
