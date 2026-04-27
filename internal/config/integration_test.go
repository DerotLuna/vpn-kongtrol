package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// TestLoad_AllAdapterTypes verifies that the validator accepts every supported
// adapter type without false-positive validation errors.
func TestLoad_AllAdapterTypes(t *testing.T) {
	types := []string{
		"forticlient", "openvpn", "protonvpn",
		"ciscoanyconnect", "wireguard", "globalprotect",
		"tailscale", "cloudflarewarp",
	}

	for _, adapterType := range types {
		t.Run(adapterType, func(t *testing.T) {
			yaml := minimalYAML(adapterType)
			path := writeTempConfig(t, yaml)

			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("Load(%s): unexpected error: %v", adapterType, err)
			}
			if cfg.VPNs["test"].Type != adapterType {
				t.Fatalf("type mismatch: got %q, want %q", cfg.VPNs["test"].Type, adapterType)
			}
		})
	}
}

// TestLoad_UnknownAdapterType ensures an unknown type is rejected.
func TestLoad_UnknownAdapterType_Fails(t *testing.T) {
	path := writeTempConfig(t, minimalYAML("magicvpn"))
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected validation error for unknown adapter type, got nil")
	}
	if !strings.Contains(err.Error(), "oneof") && !strings.Contains(err.Error(), "validation") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// TestRoundTrip writes a config with multiple profiles, loads it back,
// and verifies structural integrity.
func TestRoundTrip_MultipleProfiles(t *testing.T) {
	raw := `
vpns:
  office:
    type: forticlient
    host: vpn.empresa.com
    port: 443
    tunnel_name: "Office"
    auth:
      method: certificate+credentials
      cert: /tmp/office.crt
      key: /tmp/office.key
      username: admin
      password_keychain: office.password
    priority: 10

  wg-home:
    type: wireguard
    config: /tmp/wg0.conf
    auth:
      method: certificate
    priority: 5

  proxy:
    type: cloudflarewarp
    auth:
      method: credentials
    priority: 1
`
	path := writeTempConfig(t, raw)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.VPNs) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(cfg.VPNs))
	}
	if cfg.VPNs["office"].Host != "vpn.empresa.com" {
		t.Errorf("office.Host = %q", cfg.VPNs["office"].Host)
	}
	if cfg.VPNs["office"].Auth.Username != "admin" {
		t.Errorf("office.Auth.Username = %q", cfg.VPNs["office"].Auth.Username)
	}
	if cfg.VPNs["wg-home"].Type != "wireguard" {
		t.Errorf("wg-home.Type = %q", cfg.VPNs["wg-home"].Type)
	}
	if cfg.VPNs["proxy"].Type != "cloudflarewarp" {
		t.Errorf("proxy.Type = %q", cfg.VPNs["proxy"].Type)
	}
}

// TestDefaults_AppliedOnLoad checks that zero-value fields get sensible defaults.
func TestDefaults_AppliedOnLoad(t *testing.T) {
	path := writeTempConfig(t, minimalYAML("openvpn"))
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Monitor.Dashboard.Port != 9741 {
		t.Errorf("dashboard port default: got %d, want 9741", cfg.Monitor.Dashboard.Port)
	}
	if cfg.Monitor.Dashboard.Bind != "127.0.0.1" {
		t.Errorf("dashboard bind default: got %q, want 127.0.0.1", cfg.Monitor.Dashboard.Bind)
	}
	if cfg.Monitor.HealthCheck.Interval != "30s" {
		t.Errorf("health check interval default: got %q, want 30s", cfg.Monitor.HealthCheck.Interval)
	}
	if cfg.Security.AuditLog.MaxSizeMB != 100 {
		t.Errorf("audit log max size default: got %d, want 100", cfg.Security.AuditLog.MaxSizeMB)
	}
}

// TestLoad_HomeDirExpansion verifies that ~/.kongtrol/... paths are expanded.
func TestLoad_HomeDirExpansion(t *testing.T) {
	raw := `
vpns:
  server:
    type: openvpn
    config: ~/.kongtrol/configs/server.ovpn
    auth:
      method: certificate
      cert: ~/.kongtrol/certs/server.crt
      key: ~/.kongtrol/certs/server.key
`
	path := writeTempConfig(t, raw)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	home, _ := os.UserHomeDir()
	wantConfig := filepath.Join(home, ".kongtrol", "configs", "server.ovpn")
	if cfg.VPNs["server"].ConfigFile != wantConfig {
		t.Errorf("ConfigFile = %q, want %q", cfg.VPNs["server"].ConfigFile, wantConfig)
	}
	wantCert := filepath.Join(home, ".kongtrol", "certs", "server.crt")
	if cfg.VPNs["server"].Auth.Cert != wantCert {
		t.Errorf("Auth.Cert = %q, want %q", cfg.VPNs["server"].Auth.Cert, wantCert)
	}
}

// TestLoad_SecurityAndMonitor exercises a full config with all sections.
func TestLoad_SecurityAndMonitor(t *testing.T) {
	raw := `
vpns:
  office:
    type: tailscale
    auth:
      method: credentials
security:
  kill_switch:
    enabled: true
    mode: strict
    allow_lan: true
  dns_guard:
    enabled: true
    fallback_dns: "1.1.1.1"
  leak_detection:
    enabled: true
    interval: "60s"
    action: notify
  audit_log:
    path: /tmp/audit.log
    max_size_mb: 50
    sign: true
monitor:
  enabled: true
  dashboard:
    port: 9741
    bind: "127.0.0.1"
`
	path := writeTempConfig(t, raw)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.Security.KillSwitch.Enabled {
		t.Error("kill switch should be enabled")
	}
	if cfg.Security.KillSwitch.Mode != "strict" {
		t.Errorf("kill switch mode = %q", cfg.Security.KillSwitch.Mode)
	}
	if !cfg.Security.DNSGuard.Enabled {
		t.Error("dns guard should be enabled")
	}
	if !cfg.Security.AuditLog.Sign {
		t.Error("audit log sign should be true")
	}
	if cfg.Monitor.Dashboard.Port != 9741 {
		t.Errorf("dashboard port = %d", cfg.Monitor.Dashboard.Port)
	}
}

// TestLoad_EnvVarExpansion verifies $ENV_VAR substitution in YAML values.
func TestLoad_EnvVarExpansion_Integration(t *testing.T) {
	t.Setenv("TEST_VPN_HOST", "vpn.test.internal")
	raw := `
vpns:
  env-test:
    type: ciscoanyconnect
    host: $TEST_VPN_HOST
    auth:
      method: credentials
      username: testuser
`
	path := writeTempConfig(t, raw)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.VPNs["env-test"].Host != "vpn.test.internal" {
		t.Errorf("Host = %q, want %q", cfg.VPNs["env-test"].Host, "vpn.test.internal")
	}
}

// TestYAML_NodeRoundTrip verifies that a yaml.Node built programmatically
// (as the wizard does) marshals to valid YAML and loads correctly.
func TestYAML_NodeRoundTrip(t *testing.T) {
	// Simulate wizard output: build a mapping node manually.
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	vpnsNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	profileNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	profileNode.Content = append(profileNode.Content,
		strNode("type"), strNode("wireguard"),
		strNode("config"), strNode("/tmp/wg0.conf"),
		strNode("priority"), intStrNode("5"),
		strNode("auth"), func() *yaml.Node {
			a := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			a.Content = append(a.Content, strNode("method"), strNode("certificate"))
			return a
		}(),
	)
	vpnsNode.Content = append(vpnsNode.Content, strNode("wg-home"), profileNode)
	root.Content = append(root.Content, strNode("vpns"), vpnsNode)

	data, err := yaml.Marshal(root)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	path := writeTempConfig(t, string(data))
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load from wizard-generated YAML: %v", err)
	}
	if cfg.VPNs["wg-home"].Type != "wireguard" {
		t.Errorf("type = %q", cfg.VPNs["wg-home"].Type)
	}
	if cfg.VPNs["wg-home"].ConfigFile != "/tmp/wg0.conf" {
		t.Errorf("config = %q", cfg.VPNs["wg-home"].ConfigFile)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func minimalYAML(adapterType string) string {
	extra := ""
	switch adapterType {
	case "openvpn":
		extra = "\n    config: /tmp/test.ovpn"
	case "wireguard":
		extra = "\n    config: /tmp/wg0.conf"
	}
	return `
vpns:
  test:
    type: ` + adapterType + extra + `
    auth:
      method: credentials
`
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "kongtrol-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(strings.TrimSpace(content) + "\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func strNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"}
}

func intStrNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!int"}
}
