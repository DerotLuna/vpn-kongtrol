package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStateLoadSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	in := cliSessionState{
		LastCommandAt: time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second),
		LastCommand:   "kongtrol status",
	}
	if err := saveSessionState(path, in); err != nil {
		t.Fatalf("saveSessionState: %v", err)
	}
	out, err := loadSessionState(path)
	if err != nil {
		t.Fatalf("loadSessionState: %v", err)
	}
	if out.LastCommand != in.LastCommand {
		t.Fatalf("LastCommand=%q want %q", out.LastCommand, in.LastCommand)
	}
	if !out.LastCommandAt.Equal(in.LastCommandAt) {
		t.Fatalf("LastCommandAt=%v want %v", out.LastCommandAt, in.LastCommandAt)
	}
}
