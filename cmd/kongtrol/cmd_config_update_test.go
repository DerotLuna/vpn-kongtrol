package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestSetUpdateCheckInFile_TogglesAndKeepsComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	body := "# my VPNs\nvpns:\n  office:\n    type: openvpn # work\n    auth:\n      method: credentials\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UpdateCheckEnabled() {
		t.Fatal("update check must be off by default")
	}
	for _, on := range []bool{true, false} {
		if err := setUpdateCheckInFile(path, on); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.UpdateCheckEnabled() != on {
			t.Fatalf("want %v", on)
		}
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# my VPNs") || !strings.Contains(string(data), "# work") {
		t.Fatalf("comments lost:\n%s", data)
	}
}
