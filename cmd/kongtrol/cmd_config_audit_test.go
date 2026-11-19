package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// Without a daemon, config audit-log writes kongtrol.yaml after the same
// validation the daemon runs; a refused value leaves the file as it was.
func TestSetAuditLogInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	raw := "# my comment\nvpns:\n  office:\n    type: wireguard\n    config: /tmp/office.conf\n    auth:\n      method: credentials\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	prev := cfg
	cfg = loaded
	t.Cleanup(func() { cfg = prev })

	days, total, small := 7, 300, 50
	if err := setAuditLogInFile(path, daemonctl.AuditLogPatch{RetentionDays: &days, MaxTotalMB: &total}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	for _, want := range []string{"# my comment", "retention_days: 7", "max_total_mb: 300"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("file lacks %q:\n%s", want, got)
		}
	}
	re, err := config.Load(path)
	if err != nil || re.Security.AuditLog.RetentionDays != 7 || re.Security.AuditLog.MaxTotalMB != 300 {
		t.Fatalf("reload: %+v, %v", re.Security.AuditLog, err)
	}

	// A total below one file (default 100 MB) is refused, file unchanged.
	if err := setAuditLogInFile(path, daemonctl.AuditLogPatch{MaxTotalMB: &small}); err == nil {
		t.Fatal("total 50 MB below a 100 MB file was accepted")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(got) {
		t.Fatal("a refused value changed the file")
	}
}
