package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kongtrol.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

const validConfig = `
vpns:
  office:
    type: forticlient
    host: vpn.example.com
    port: 443
    tunnel_name: Office
    auth:
      method: certificate+credentials
      cert: /tmp/cert.crt
      key: /tmp/cert.key
    priority: 10

  server:
    type: openvpn
    config: /tmp/server.ovpn
    auth:
      method: certificate
    priority: 20

policies:
  - name: "Office"
    match:
      ip_ranges: ["10.10.0.0/16"]
    via: office

security:
  kill_switch:
    enabled: true
    mode: strict

monitor:
  enabled: true
  dashboard:
    port: 9741
    bind: "127.0.0.1"
`

func TestLoad_ValidConfig(t *testing.T) {
	path := writeConfig(t, validConfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.VPNs) != 2 {
		t.Errorf("VPNs count = %d, want 2", len(cfg.VPNs))
	}
	if cfg.VPNs["office"].Type != "forticlient" {
		t.Errorf("office.Type = %q, want forticlient", cfg.VPNs["office"].Type)
	}
	if len(cfg.Policies) != 1 {
		t.Errorf("Policies count = %d, want 1", len(cfg.Policies))
	}
	if cfg.VPNs["office"].AllowInsecureCert {
		t.Errorf("office.AllowInsecureCert = true, want false by default")
	}
}

func TestLoad_FortiClientAllowInsecureCert(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: forticlient
    allow_insecure_cert: true
    host: vpn.example.com
    port: 443
    tunnel_name: Office
    auth:
      method: certificate+credentials
      cert: /tmp/cert.crt
      key: /tmp/cert.key
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.VPNs["office"].AllowInsecureCert {
		t.Errorf("office.AllowInsecureCert = false, want true")
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, validConfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Dashboard defaults
	if cfg.Monitor.Dashboard.Port != 9741 {
		t.Errorf("Dashboard.Port = %d, want 9741", cfg.Monitor.Dashboard.Port)
	}
	if cfg.Monitor.Dashboard.Bind != "127.0.0.1" {
		t.Errorf("Dashboard.Bind = %q, want 127.0.0.1", cfg.Monitor.Dashboard.Bind)
	}
	// HealthCheck defaults
	if cfg.Monitor.HealthCheck.Interval != "30s" {
		t.Errorf("HealthCheck.Interval = %q, want 30s", cfg.Monitor.HealthCheck.Interval)
	}
	// KillSwitch mode default
	if cfg.Security.KillSwitch.Mode != "strict" {
		t.Errorf("KillSwitch.Mode = %q, want strict", cfg.Security.KillSwitch.Mode)
	}
}

func TestLoad_NoVPNs_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns: {}
policies: []
`)
	_, err := Load(path)
	if err == nil {
		t.Error("expected validation error for empty vpns map, got nil")
	}
}

func TestLoad_InvalidVPNType_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  bad:
    type: not-a-real-vpn
    auth:
      method: certificate
`)
	_, err := Load(path)
	if err == nil {
		t.Error("expected validation error for unknown VPN type, got nil")
	}
}

func TestLoad_FileMissing(t *testing.T) {
	_, err := Load("/nonexistent/path/kongtrol.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "not: valid: yaml: [")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EnvExpansion(t *testing.T) {
	t.Setenv("TEST_VPN_HOST", "vpn.from-env.com")
	path := writeConfig(t, `
vpns:
  env-test:
    type: openvpn
    host: "$TEST_VPN_HOST"
    config: /tmp/test.ovpn
    auth:
      method: certificate
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.VPNs["env-test"].Host != "vpn.from-env.com" {
		t.Errorf("Host = %q, want vpn.from-env.com", cfg.VPNs["env-test"].Host)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct {
		in   string
		want string
	}{
		{"~/certs/cert.crt", filepath.Join(home, "certs/cert.crt")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}
	for _, tc := range cases {
		got := expandHome(tc.in, home)
		if got != tc.want {
			t.Errorf("expandHome(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
