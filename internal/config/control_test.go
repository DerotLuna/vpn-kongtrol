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

// Every daemon start writes a new token (ADR-0062); clients read the
// current one.
func TestRotateAPIToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	first, err := LoadOrCreateAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := RotateAPIToken()
	if err != nil || rotated == first || rotated == "" {
		t.Fatalf("rotate: %q %v (first %q)", rotated, err, first)
	}
	if cur, err := CurrentAPIToken(); err != nil || cur != rotated {
		t.Fatalf("current=%q %v, want %q", cur, err, rotated)
	}
	if again, _ := LoadOrCreateAPIToken(); again != rotated {
		t.Fatalf("LoadOrCreate after rotate=%q, want %q", again, rotated)
	}
	info, err := os.Stat(filepath.Join(home, ".kongtrol", "api-token"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions=%o, want 600", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Join(home, ".kongtrol"))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
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
