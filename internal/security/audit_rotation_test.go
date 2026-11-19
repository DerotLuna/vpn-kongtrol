package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// max_size_mb: when the next entry would pass the size, the same day goes
// on in "<base>-DATE.1<ext>", ".2"…; nothing is lost and every file stays
// under the limit (a file always takes its first line).
func TestAuditLogger_SizeRotationWithinDay(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	l, err := NewAuditLogger(base, AuditLoggerOptions{MaxSizeBytes: 600})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		if err := l.Log("INFO", "vpn.connect", "office", strings.Repeat("x", 60)); err != nil {
			t.Fatal(err)
		}
	}
	_ = l.Close()

	files := AuditDayFiles(base, time.Now())
	if len(files) < 3 {
		t.Fatalf("expected several segments, got %v", files)
	}
	if files[0] != DailyAuditLogPath(base, time.Now()) {
		t.Fatalf("first file %q is not the day file", files[0])
	}
	total := 0
	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Size() > 600 {
			t.Errorf("%s is %d bytes, over the limit", f, fi.Size())
		}
		evs, _ := ReadAuditEvents(f)
		total += len(evs)
	}
	if total != 12 {
		t.Fatalf("read %d events, want 12", total)
	}

	// Reopening continues in the newest segment, not the day file.
	l2, err := NewAuditLogger(base, AuditLoggerOptions{MaxSizeBytes: 600})
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()
	if want := len(files) - 1; l2.seg != want {
		t.Fatalf("reopened at segment %d, want %d", l2.seg, want)
	}
}

// retention_days keeps exactly the last N days (segments included).
func TestAuditLogger_RetentionDays(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	var kept, gone []string
	for d := 0; d < 6; d++ {
		day := now.AddDate(0, 0, -d).Format("2006-01-02")
		for seg := 0; seg < 2; seg++ {
			p := auditSegmentPath(base, day, seg)
			if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if d < 4 {
				kept = append(kept, p)
			} else {
				gone = append(gone, p)
			}
		}
	}
	l := &AuditLogger{basePath: base, retention: 4}
	l.pruneLocked(now)
	for _, p := range kept {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s should be kept: %v", p, err)
		}
	}
	for _, p := range gone {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be deleted", p)
		}
	}
}

// Segments are the logger's files too (purge, readers); look-alikes are not.
func TestAuditLogFiles_IncludesSegmentsOnly(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"audit-2026-11-18.log", "audit-2026-11-18.1.log", "audit-2026-11-18.10.log",
		"audit-2026-11-18.01.log", "audit-2026-11-18.0.log", "audit-2026-11-18.x.log", "audit-2026-11-18.-1.log", "audit.log"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := AuditLogFiles(filepath.Join(dir, "audit.log"))
	var names []string
	for _, p := range got {
		names = append(names, filepath.Base(p))
	}
	want := []string{"audit-2026-11-18.log", "audit-2026-11-18.1.log", "audit-2026-11-18.10.log"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %v, want %v (write order)", names, want)
	}
}

func writeEvents(t *testing.T, path string, evs ...AuditEvent) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, ev := range evs {
		b, _ := json.Marshal(ev)
		_, _ = f.Write(append(b, '\n'))
	}
}

// QueryAuditEvents reads every retained day, filters by date, and pages
// with a Before cursor without gaps or repeats.
func TestQueryAuditEvents_RangeAndPaging(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	day0 := time.Date(2026, 11, 10, 12, 0, 0, 0, time.Local)
	var all []AuditEvent
	for d := 0; d < 5; d++ {
		at := day0.AddDate(0, 0, d)
		for i := 0; i < 3; i++ {
			ev := AuditEvent{ID: at.Format("02") + "-" + string(rune('a'+i)), Timestamp: at.Add(time.Duration(i) * time.Minute).UTC(), Level: "INFO", Profile: "office"}
			if i == 2 {
				ev.Level, ev.Profile = "WARN", "home"
			}
			all = append(all, ev)
			// one event per segment file, to read across segments
			writeEvents(t, auditSegmentPath(base, at.Format("2006-01-02"), i), ev)
		}
	}

	// Range: days 11–12 only.
	from, _ := time.ParseInLocation("2006-01-02", "2026-11-11", time.Local)
	to := from.AddDate(0, 0, 2).Add(-time.Nanosecond)
	got, more, err := QueryAuditEvents(base, AuditQuery{From: from, To: to, Limit: 100})
	if err != nil || more || len(got) != 6 {
		t.Fatalf("range: %d events, more=%v, err=%v", len(got), more, err)
	}
	for _, ev := range got {
		if ev.Timestamp.Before(from) || ev.Timestamp.After(to) {
			t.Fatalf("out of range: %v", ev.Timestamp)
		}
	}

	// Filters.
	got, _, _ = QueryAuditEvents(base, AuditQuery{Level: "warn", Profile: "HOME"})
	if len(got) != 5 {
		t.Fatalf("level+profile filter: %d, want 5", len(got))
	}

	// Paging: 4 per page, newest first, all 15 exactly once.
	var seen []string
	var before time.Time
	for page := 0; page < 10; page++ {
		evs, more, err := QueryAuditEvents(base, AuditQuery{Before: before, Limit: 4})
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(evs); i++ {
			if evs[i].Timestamp.After(evs[i-1].Timestamp) {
				t.Fatal("not newest first")
			}
		}
		for _, ev := range evs {
			seen = append(seen, ev.ID)
		}
		if !more {
			break
		}
		before = evs[len(evs)-1].Timestamp
	}
	var want []string
	for _, ev := range all {
		want = append(want, ev.ID)
	}
	sort.Strings(seen)
	sort.Strings(want)
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("paging saw %v, want %v", seen, want)
	}
}

// Verify: only a signed entry with a valid MAC from this key passes.
func TestAuditLogger_Verify(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	l, path := tempAuditLog(t, true, key)
	l.Info("vpn.connect", "office", "ok")
	ev := readEvents(t, path)[0]
	if !l.Verify(ev) {
		t.Fatal("valid entry not verified")
	}
	ev.Message = "edited"
	if l.Verify(ev) {
		t.Fatal("tampered entry verified")
	}
	ev.HMAC = ""
	if l.Verify(ev) {
		t.Fatal("unsigned entry must not count as verified")
	}
	other, _ := tempAuditLog(t, true, []byte("another-key-another-key-another!!"))
	if other.Verify(readEvents(t, path)[0]) {
		t.Fatal("verified with the wrong key")
	}
}
