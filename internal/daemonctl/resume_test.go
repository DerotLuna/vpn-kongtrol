package daemonctl

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestResume_WriteThenTakeOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run", "resume.json")
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

	if err := WriteResume(path, []string{"office", "", "home", "office"}, now); err != nil {
		t.Fatal(err)
	}
	got, err := TakeResume(path, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"home", "office"}; !slices.Equal(got, want) {
		t.Fatalf("profiles=%v, want %v", got, want)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("resume file still there after TakeResume (err=%v)", err)
	}
	if again, _ := TakeResume(path, now); len(again) != 0 {
		t.Fatalf("second TakeResume=%v, want nothing (one-shot)", again)
	}
}

func TestResume_EmptyListRemovesOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.json")
	now := time.Now()
	if err := WriteResume(path, []string{"office"}, now); err != nil {
		t.Fatal(err)
	}
	if err := WriteResume(path, nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale resume file kept (err=%v)", err)
	}
	// No file at all is not an error either.
	if err := WriteResume(path, nil, now); err != nil {
		t.Fatal(err)
	}
}

func TestResume_IgnoresStaleAndBroken(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

	old := filepath.Join(dir, "old.json")
	if err := WriteResume(old, []string{"office"}, now.Add(-ResumeMaxAge-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, err := TakeResume(old, now); err != nil || len(got) != 0 {
		t.Fatalf("stale file: profiles=%v err=%v, want none", got, err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("stale file not deleted")
	}

	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := TakeResume(broken, now); err != nil || len(got) != 0 {
		t.Fatalf("broken file: profiles=%v err=%v, want none", got, err)
	}

	future := filepath.Join(dir, "v2.json")
	if err := os.WriteFile(future, []byte(`{"version":2,"profiles":["x"],"saved_at":"2026-11-19T10:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := TakeResume(future, now); len(got) != 0 {
		t.Fatalf("unknown version: profiles=%v, want none", got)
	}

	if got, err := TakeResume(filepath.Join(dir, "missing.json"), now); err != nil || got != nil {
		t.Fatalf("missing file: profiles=%v err=%v", got, err)
	}
}
