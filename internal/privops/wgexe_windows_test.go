//go:build windows

package privops

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// A folder a standard user created (owner = that user, user can write) is
// not trusted for a binary the helper runs as SYSTEM.
func TestAdminOnlyWritable_RejectsUserFolder(t *testing.T) {
	dir := t.TempDir() // owned by the test user, writable by them
	if err := adminOnlyWritable(dir); err == nil {
		t.Skip("running elevated: temp dir owned by Administrators")
	}
	f := filepath.Join(dir, "wg.exe")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := adminOnlyWritable(f); err == nil {
		t.Fatal("user-owned file accepted")
	}
}

// Regression: the constant had two wrong sub-authorities, so files owned by
// TrustedInstaller (most of System32 / Program Files) were not trusted.
func TestTrustedInstallerSID_MatchesSystem(t *testing.T) {
	sid, _, _, err := windows.LookupSID("", `NT SERVICE\TrustedInstaller`)
	if err != nil {
		t.Skip(err)
	}
	if sid.String() != trustedInstallerSID {
		t.Fatalf("trustedInstallerSID = %s, system says %s", trustedInstallerSID, sid.String())
	}
}

// The real WireGuard install (when present) passes: Program Files ACLs
// give Users read/execute only.
func TestWGExe_RealInstall(t *testing.T) {
	if _, err := os.Stat(filepath.Join(programFiles(), "WireGuard", "wg.exe")); err != nil {
		t.Skip("WireGuard not installed")
	}
	if _, err := wgExe("wg.exe"); err != nil {
		t.Fatalf("installed wg.exe rejected: %v", err)
	}
	if _, err := wgExe("wireguard.exe"); err != nil {
		t.Fatalf("installed wireguard.exe rejected: %v", err)
	}
}
