package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestNewPublicIPChecker_OffWithKillSwitchAndDNSGuard(t *testing.T) {
	c := &config.Config{}
	c.Security.KillSwitch.Enabled = true
	c.Security.DNSGuard.Enabled = true
	lt := newPublicIPChecker(c)
	if lt.Enabled() {
		t.Fatal("kill switch / DNS guard must not turn the public IP lookup on")
	}
	if r := lt.CheckNow(); !r.Disabled {
		t.Errorf("CheckNow = %+v, want Disabled (no request)", r)
	}
}

func TestSetPublicIPInFile_KeepsCommentsAndClearsLegacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	body := "# my config\nvpns:\n  office:\n    type: openvpn\n    auth:\n      method: credentials\nsecurity:\n  leak_detection:\n    enabled: true # old\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := setPublicIPInFile(path, false); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicIPCheckEnabled() {
		t.Error("off should also clear leak_detection.enabled")
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# my config") {
		t.Error("comments lost")
	}
	if err := setPublicIPInFile(path, true); err != nil {
		t.Fatal(err)
	}
	if c, _ = config.Load(path); !c.Security.PublicIPCheck.Enabled {
		t.Error("on not written")
	}
}
