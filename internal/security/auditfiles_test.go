package security

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// Purge deletes only the files the logger writes (base-YYYY-MM-DD.ext),
// never the configured base path or look-alikes.
func TestAuditLogFiles_OnlyDailyFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"audit.log", "audit-2026-11-17.log", "audit-2026-11-18.log", "audit-notes.log", "audit-2026-11-18.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := AuditLogFiles(filepath.Join(dir, "audit.log"))
	sort.Strings(got)
	want := []string{filepath.Join(dir, "audit-2026-11-17.log"), filepath.Join(dir, "audit-2026-11-18.log")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if AuditLogFiles("") != nil {
		t.Fatal("empty base must list nothing")
	}
}
