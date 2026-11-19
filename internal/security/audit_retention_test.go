package security

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock drives the audit logger's clock and its midnight timer.
type fakeClock struct {
	mu    sync.Mutex
	t     time.Time
	fire  chan time.Time
	waits chan time.Duration
}

func newFakeClock(t time.Time) *fakeClock {
	return &fakeClock{t: t, fire: make(chan time.Time), waits: make(chan time.Duration, 4)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) set(t time.Time) {
	c.mu.Lock()
	c.t = t
	c.mu.Unlock()
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.waits <- d
	return c.fire
}

func (c *fakeClock) nextWait(t *testing.T) time.Duration {
	t.Helper()
	select {
	case d := <-c.waits:
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("the cleanup loop did not wait for midnight")
		return 0
	}
}

// The cleanup runs every local midnight without any new entry: with
// retention_days 1, yesterday's file goes as soon as the day changes.
func TestAuditLogger_MidnightCleanup(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	day1 := time.Date(2026, 11, 18, 23, 0, 0, 0, time.Local)
	clk := newFakeClock(day1)
	l, err := NewAuditLogger(base, AuditLoggerOptions{RetentionDays: 1, Now: clk.Now, After: clk.After})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if err := l.Log("INFO", "vpn.connect", "office", "ok"); err != nil {
		t.Fatal(err)
	}
	if d := clk.nextWait(t); d != time.Hour+time.Second {
		t.Fatalf("waits %v for midnight, want 1h0m1s", d)
	}
	yesterday := DailyAuditLogPath(base, day1)
	if _, err := os.Stat(yesterday); err != nil {
		t.Fatalf("today's file missing before midnight: %v", err)
	}

	clk.set(time.Date(2026, 11, 19, 0, 0, 1, 0, time.Local))
	clk.fire <- clk.Now()
	if d := clk.nextWait(t); d != 24*time.Hour {
		t.Fatalf("next wait %v, want 24h", d)
	}
	if _, err := os.Stat(yesterday); !os.IsNotExist(err) {
		t.Fatalf("yesterday's file should be gone after midnight, err=%v", err)
	}
	// Logging goes on in the new day's file.
	if err := l.Log("INFO", "vpn.disconnect", "office", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(DailyAuditLogPath(base, clk.Now())); err != nil {
		t.Fatalf("new day's file: %v", err)
	}
}

// max_total_mb: the oldest files go first so all files, with room for the
// open one to fill up, stay under the total; the open file is never deleted.
func TestAuditLogger_TotalCapDeletesOldestFirst(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "audit.log")
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.Local)
	// Five older days of 300 bytes each.
	var days []string
	for d := 5; d >= 1; d-- {
		p := DailyAuditLogPath(base, now.AddDate(0, 0, -d))
		if err := os.WriteFile(p, []byte(strings.Repeat("x", 299)+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		days = append(days, p)
	}
	// Total 1000, one file up to 300: the older files may use 700, so only
	// the two newest days stay.
	clk := newFakeClock(now)
	l, err := NewAuditLogger(base, AuditLoggerOptions{
		MaxSizeBytes: 300, MaxTotalBytes: 1000, Now: clk.Now, After: clk.After,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for i, p := range days {
		_, err := os.Stat(p)
		if i < 3 && !os.IsNotExist(err) {
			t.Errorf("%s (old) should be deleted", filepath.Base(p))
		}
		if i >= 3 && err != nil {
			t.Errorf("%s (recent) should be kept: %v", filepath.Base(p), err)
		}
	}

	// Writing past a file's size opens a segment and prunes again; the
	// total never passes the cap.
	for i := 0; i < 40; i++ {
		if err := l.Log("INFO", "vpn.connect", "office", strings.Repeat("y", 100)); err != nil {
			t.Fatal(err)
		}
		var total int64
		for _, p := range AuditLogFiles(base) {
			fi, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			total += fi.Size()
		}
		if total > 1000 {
			t.Fatalf("after %d entries the audit files use %d bytes, over 1000", i+1, total)
		}
	}
	if files := AuditLogFiles(base); len(files) == 0 || files[len(files)-1] != auditSegmentPath(base, now.Format(auditDateLayout), l.seg) {
		t.Fatalf("the open file must stay: %v", files)
	}
}

func TestEffectiveAuditRetentionDays(t *testing.T) {
	for in, want := range map[int]int{-1: 30, 0: 30, 1: 1, 30: 30, 365: 365, 400: 365} {
		if got := EffectiveAuditRetentionDays(in); got != want {
			t.Errorf("EffectiveAuditRetentionDays(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestUntilNextMidnight(t *testing.T) {
	at := time.Date(2026, 11, 19, 22, 30, 0, 0, time.Local)
	if d := untilNextMidnight(at); d != 90*time.Minute+time.Second {
		t.Fatalf("got %v", d)
	}
}
