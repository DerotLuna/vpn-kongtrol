package vpndetect

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileVersionMissingFile(t *testing.T) {
	if v := FileVersion(filepath.Join(t.TempDir(), "nope.exe")); v != "" {
		t.Fatalf("missing file: got %q", v)
	}
}

func TestFileVersionOwnBinary(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PE version resources exist only on Windows")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	// A go test binary has no version resource: must answer "" quickly,
	// never launch it.
	_ = FileVersion(exe)
}
