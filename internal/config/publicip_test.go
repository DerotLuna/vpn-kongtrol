package config

import "testing"

func TestValidatePublicIPEndpoint(t *testing.T) {
	ok := []string{"https://api.ipify.org", "https://icanhazip.com/", "https://checkip.amazonaws.com", "https://example.com:8443/ip?format=text"}
	bad := []string{"http://api.ipify.org", "api.ipify.org", "https://", "https://user:pw@example.com", "ftp://example.com", "", "https:example.com"}
	for _, u := range ok {
		if err := ValidatePublicIPEndpoint(u); err != nil {
			t.Errorf("%q rejected: %v", u, err)
		}
	}
	for _, u := range bad {
		if err := ValidatePublicIPEndpoint(u); err == nil {
			t.Errorf("%q accepted", u)
		}
	}
}

func TestPublicIPCheckDefaults(t *testing.T) {
	c := &Config{}
	c.Security.KillSwitch.Enabled = true
	c.Security.DNSGuard.Enabled = true
	if c.PublicIPCheckEnabled() {
		t.Error("kill switch / DNS guard must not enable the public IP lookup")
	}
	if got := c.PublicIPHost(); got != "api.ipify.org" {
		t.Errorf("default host = %q", got)
	}
	c.Security.LeakDetection.Enabled = true
	if !c.PublicIPCheckEnabled() {
		t.Error("legacy leak_detection.enabled should opt in")
	}
	c.SetPublicIPCheck(false)
	if c.PublicIPCheckEnabled() {
		t.Error("SetPublicIPCheck(false) should also clear the legacy flag")
	}
}

// ADR-0041: leak_detection.interval/action were removed; an existing
// config that still has them must keep loading (enabled still opts in).
func TestLoad_LegacyLeakDetectionKeysStillLoad(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: wireguard
    auth:
      method: certificate
security:
  leak_detection:
    enabled: true
    interval: 120s
    action: killswitch_and_notify
`)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("legacy keys broke loading: %v", err)
	}
	if !c.PublicIPCheckEnabled() {
		t.Error("leak_detection.enabled should still opt in")
	}
}
