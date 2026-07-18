package monitor

import (
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestTimeWindowMatch_DayAndOvernight(t *testing.T) {
	now := time.Date(2026, 7, 14, 10, 30, 0, 0, time.UTC)
	if !timeWindowMatch("09:00", "18:00", now) {
		t.Fatal("expected daytime window to match")
	}
	if timeWindowMatch("18:00", "09:00", now) {
		t.Fatal("expected overnight window not to match at 10:30")
	}
	night := time.Date(2026, 7, 14, 23, 30, 0, 0, time.UTC)
	if !timeWindowMatch("22:00", "06:00", night) {
		t.Fatal("expected overnight window to match at 23:30")
	}
}

func TestRuleActiveNow_WeekdayAndWindow(t *testing.T) {
	r := config.ScheduleRule{
		Weekdays: []string{"mon", "tue", "wed", "thu", "fri"},
		Start:    "09:00",
		End:      "18:00",
	}
	ts := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC) // Tue
	if !ruleActiveNow(r, ts) {
		t.Fatal("expected business-hours rule active on Tuesday 10:00")
	}
	off := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC) // Sun
	if ruleActiveNow(r, off) {
		t.Fatal("expected rule inactive on Sunday")
	}
}
