package main

import (
	"strings"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/api"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// Audit entries the daemon writes besides the tunnel lifecycle
// (ProfileService) and the watchdog (daemon_watchdog.go): API requests,
// refused credentials, daemon start/stop and kill switch state changes.
// Inventory in docs/SECURITY.md ("Qué queda en el registro de auditoría",
// ADR-0064). Messages never carry secrets: no bodies, tokens, passwords,
// keys or file contents.

// auditAPIRequest renders an API audit record in the daemon's language.
func auditAPIRequest(rec api.AuditRecord) {
	logAudit(rec.Level, rec.Action, rec.Profile, apiAuditMessage(rec))
}

func apiAuditMessage(rec api.AuditRecord) string {
	if rec.Action == "auth.denied" {
		msg := cf("cli.audit.auth.denied", rec.Method, rec.Path, ct("cli.audit.auth.reason."+rec.Reason))
		if rec.Suppressed > 0 {
			msg += " " + cf("cli.audit.auth.skipped", rec.Suppressed)
		}
		return msg
	}
	who := ct("cli.audit.client." + rec.Client)
	if rec.Name != "" {
		who += " · " + rec.Name
	}
	switch {
	case rec.Status == 0:
		return cf("cli.audit.api.started", who)
	case rec.Status >= 400:
		return cf("cli.audit.api.refused", who, rec.Status)
	}
	return cf("cli.audit.api.done", who, rec.Status)
}

// auditDaemonStart / auditDaemonStop mark the daemon's lifetime, so a gap
// in the log (nothing recorded while it was off) is visible.
func auditDaemonStart() {
	logAudit("INFO", "daemon.start", "", cf("cli.audit.daemon.start", version))
}

func auditDaemonStop() {
	logAudit("INFO", "daemon.stop", "", ct("cli.audit.daemon.stop"))
}

var (
	ksAuditMu    sync.Mutex
	ksAuditState string
	ksAuditErr   string
)

// auditKillSwitchState writes an entry when the kill switch changes state
// (off / armed / active, with what it blocks) or fails to apply its block.
func auditKillSwitchState(st security.KillSwitchStatus) {
	key, args := st.StateKey()
	state := st.State + "|" + strings.Join(st.Blocking, ",") + "|" + strings.Join(st.PolicyBlocking, ",")
	ksAuditMu.Lock()
	changed := state != ksAuditState
	failed := st.Error != "" && st.Error != ksAuditErr
	ksAuditState, ksAuditErr = state, st.Error
	ksAuditMu.Unlock()
	if changed {
		level := "SECURITY"
		if st.State != security.KillSwitchActive {
			level = "INFO"
		}
		msg := cf(key, args...)
		if len(st.PolicyBlocking) > 0 {
			msg += " · " + cf("cli.audit.killswitch.policies", strings.Join(st.PolicyBlocking, ", "))
		}
		logAudit(level, "security.killswitch.state", strings.Join(st.Blocking, ","), msg)
	}
	if failed {
		logAudit("ERROR", "security.killswitch.error", "", cf("cli.audit.killswitch.error", st.Error))
	}
}
