package privhelper

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

const (
	auditMaxBytes = 5 << 20
	auditMaxArgs  = 2048
)

// AuditEntry is one line of the helper audit log (JSON lines).
type AuditEntry struct {
	Time     time.Time       `json:"time"`
	Caller   string          `json:"caller"`
	CallerID string          `json:"caller_id"`
	Op       string          `json:"op"`
	Args     json.RawMessage `json:"args,omitempty"`
	OK       bool            `json:"ok"`
	Code     string          `json:"code,omitempty"`
	Error    string          `json:"error,omitempty"`
	Millis   int64           `json:"ms"`
}

// Audit appends entries to a helper-owned log file (0600 / admin-only dir),
// rotating once to <path>.1 past 5 MB. A nil *Audit discards entries.
type Audit struct {
	mu   sync.Mutex
	path string
}

// OpenAudit returns an audit log writing to path.
func OpenAudit(path string) *Audit { return &Audit{path: path} }

// Record writes e. Results are never logged (they may hold secrets such as
// the OpenVPN management password); args never contain secrets.
func (a *Audit) Record(e AuditEntry) {
	if a == nil {
		return
	}
	if len(e.Args) > auditMaxArgs || !json.Valid(e.Args) {
		e.Args = nil
	}
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if fi, err := os.Stat(a.path); err == nil && fi.Size() > auditMaxBytes {
		_ = os.Rename(a.path, a.path+".1")
	}
	f, err := os.OpenFile(a.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
	_ = f.Close()
}
