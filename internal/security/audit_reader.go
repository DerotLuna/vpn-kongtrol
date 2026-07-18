package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ReadAuditEvents reads and parses a single audit log file, skipping any
// malformed lines. A missing file is not an error — it returns an empty
// slice, since a profile with no activity yet simply has no log file.
func ReadAuditEvents(path string) ([]AuditEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEvent{}, nil
		}
		return nil, fmt.Errorf("audit: open %s: %w", path, err)
	}
	defer f.Close()

	var out []AuditEvent
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev AuditEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("audit: read %s: %w", path, err)
	}
	return out, nil
}

// ReadRecentAuditEvents reads audit events from the last `days` daily-rotated
// log files (including today), oldest day first. Shared by the CLI `logs`
// command and the dashboard's GET /api/v1/audit endpoint.
func ReadRecentAuditEvents(basePath string, days int) ([]AuditEvent, error) {
	if basePath == "" {
		return []AuditEvent{}, nil
	}
	if days <= 0 {
		days = 1
	}
	out := make([]AuditEvent, 0, 256)
	for i := days - 1; i >= 0; i-- {
		path := DailyAuditLogPath(basePath, time.Now().AddDate(0, 0, -i))
		evs, err := ReadAuditEvents(path)
		if err != nil {
			return nil, err
		}
		out = append(out, evs...)
	}
	return out, nil
}
