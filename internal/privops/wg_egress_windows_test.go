package privops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsDefaultEgressPreservesSpecificRoutes(t *testing.T) {
	fixture := strings.Replace(dualStackConfig, "::/0", "0:0:0:0:0:0:0:0/0, 10.11.0.0/16 # specific", 1)
	fixture = strings.Replace(fixture, "[Peer]", "SaveConfig = false\n[Peer]", 1)
	path := filepath.Join(t.TempDir(), "test.conf")
	if err := os.WriteFile(path, []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	if got := CheckDefaultEgress("wireguard", path, ""); !got.Compatible {
		t.Fatalf("check = %+v", got)
	}
	data := []byte(fixture)
	got := string(prepareWireGuardDefaultEgress(data))
	allowed := "AllowedIPs = 0.0.0.0/1, 128.0.0.0/1, ::/1, 8000::/1, 10.11.0.0/16"
	if !strings.Contains(got, allowed) || strings.Contains(got, "SaveConfig") || string(data) != fixture {
		t.Fatal("the private copy must preserve specific routes without changing the original")
	}
}

func TestConflictingWindowsDefaultRoutes(t *testing.T) {
	for _, tt := range []struct {
		output, replacing string
		conflict          bool
	}{
		{},
		{"4|0.0.0.0/1|WG\r\n4|128.0.0.0/1|WG", "wg", false},
		{"5|::/0|Other VPN", "wg", true},
		{"5|128.0.0.0/1|Other VPN", "", true},
		{"5|192.0.2.0/24|Other VPN", "", false},
	} {
		if got := conflictingWindowsDefaultRoutes(tt.output, tt.replacing); got != tt.conflict {
			t.Errorf("conflict for %q replacing %q = %t", tt.output, tt.replacing, got)
		}
	}
}
