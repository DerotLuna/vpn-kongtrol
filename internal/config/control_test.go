package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateDashboardBind(t *testing.T) {
	for _, bind := range []string{"127.0.0.1", "::1", "localhost"} {
		if err := ValidateDashboardBind(bind); err != nil {
			t.Errorf("ValidateDashboardBind(%q): %v", bind, err)
		}
	}
	for _, bind := range []string{"0.0.0.0", "192.168.1.10", "example.com", ""} {
		if err := ValidateDashboardBind(bind); err == nil {
			t.Errorf("ValidateDashboardBind(%q) succeeded, want error", bind)
		}
	}
}

func TestWriteFileAtomic_ReplacesContentsAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("contents=%q, want new", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions=%o, want 600", info.Mode().Perm())
	}
}
