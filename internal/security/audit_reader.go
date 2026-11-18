package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// maxAuditLine bounds one audit line (an entry is a few hundred bytes; a
// longer line is not one the logger wrote and is skipped).
const maxAuditLine = 1 << 20

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
	sc.Buffer(make([]byte, 0, 64*1024), maxAuditLine)
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

// AuditQuery selects audit events. Zero values mean "no bound".
type AuditQuery struct {
	From, To time.Time // inclusive bounds on Timestamp
	Before   time.Time // exclusive upper bound: the page cursor
	Profile  string    // case-insensitive exact match
	Level    string    // case-insensitive exact match
	Limit    int       // max events returned (≤ 0: DefaultAuditQueryLimit)
}

const (
	// DefaultAuditQueryLimit / MaxAuditQueryLimit bound one page.
	DefaultAuditQueryLimit = 200
	MaxAuditQueryLimit     = 1000
)

// QueryAuditEvents returns the events matching q, newest first, and
// whether older matching events remain (ask again with Before set to the
// last returned Timestamp). It reads only the days the range covers, newest
// day first, and stops as soon as the page is full. Shared by the CLI
// (`kongtrol logs`) and GET /api/v1/audit.
func QueryAuditEvents(basePath string, q AuditQuery) ([]AuditEvent, bool, error) {
	if q.Limit <= 0 {
		q.Limit = DefaultAuditQueryLimit
	}
	if q.Limit > MaxAuditQueryLimit {
		q.Limit = MaxAuditQueryLimit
	}
	upper := q.To
	if !q.Before.IsZero() && (upper.IsZero() || q.Before.Before(upper)) {
		upper = q.Before
	}
	var fromKey, toKey string
	if !q.From.IsZero() {
		fromKey = q.From.In(time.Local).Format(auditDateLayout)
	}
	if !upper.IsZero() {
		toKey = upper.In(time.Local).Format(auditDateLayout)
	}

	// Group the files by day, newest day first.
	byDay := map[string][]auditFile{}
	var days []string
	for _, af := range listAuditFiles(basePath) {
		if (fromKey != "" && af.Date < fromKey) || (toKey != "" && af.Date > toKey) {
			continue
		}
		if _, ok := byDay[af.Date]; !ok {
			days = append(days, af.Date)
		}
		byDay[af.Date] = append(byDay[af.Date], af)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(days)))

	out := make([]AuditEvent, 0, min(q.Limit+1, 256))
	for _, day := range days {
		var dayEvents []AuditEvent
		for _, af := range byDay[day] {
			evs, err := ReadAuditEvents(af.Path)
			if err != nil {
				return nil, false, err
			}
			for _, ev := range evs {
				if q.matches(ev) {
					dayEvents = append(dayEvents, ev)
				}
			}
		}
		sort.SliceStable(dayEvents, func(i, j int) bool { return dayEvents[i].Timestamp.After(dayEvents[j].Timestamp) })
		out = append(out, dayEvents...)
		if len(out) > q.Limit {
			break // a full page plus one: older events remain
		}
	}
	more := len(out) > q.Limit
	if more {
		out = out[:q.Limit]
	}
	return out, more, nil
}

func (q AuditQuery) matches(ev AuditEvent) bool {
	if q.Profile != "" && !strings.EqualFold(ev.Profile, q.Profile) {
		return false
	}
	if q.Level != "" && !strings.EqualFold(ev.Level, q.Level) {
		return false
	}
	if !q.From.IsZero() && ev.Timestamp.Before(q.From) {
		return false
	}
	if !q.To.IsZero() && ev.Timestamp.After(q.To) {
		return false
	}
	if !q.Before.IsZero() && !ev.Timestamp.Before(q.Before) {
		return false
	}
	return true
}
