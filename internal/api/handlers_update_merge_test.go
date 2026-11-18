package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// mergeTestYAML holds a profile with every field the dashboard form does not
// expose, plus a policy, group, scheduler rule and settings, so update tests
// can prove the partial-update handlers keep what the request omits.
const mergeTestYAML = `
vpns:
  office:
    type: openvpn
    version: "2.6"
    host: vpn.example.com
    port: 1194
    tunnel_name: office-tun
    allow_insecure_cert: true
    config: /etc/kongtrol/office.ovpn
    server: US
    protocol: udp
    binary_path: /opt/openvpn/sbin/openvpn
    priority: 20
    kill_switch: false
    auth:
      method: certificate
      cert: /etc/kongtrol/office.crt
      key: /etc/kongtrol/office.key
  backup:
    type: wireguard
    auth:
      method: certificate
groups:
  work:
    profiles: [office, backup]
policies:
  - name: corp
    via: office
    match:
      ip_ranges: ["10.0.0.0/8"]
      domains: ["*.corp.example"]
      apps: ["outlook.exe"]
security:
  leak_detection:
    action: notify
  audit_log:
    path: /var/log/kongtrol/audit.log
    sign: true
monitor:
  scheduler:
    rules:
      - name: weekdays
        profiles: [office]
        weekdays: [mon, tue]
        start: "08:00"
        end: "18:00"
`

func doJSON(t *testing.T, s *Server, method, target, pattern string, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(method+" "+pattern, h)
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func reload(t *testing.T, path string) *config.Config {
	t.Helper()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestUpdateVPN_PartialBodyKeepsUnsentFields(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleUpdateVPN,
		`{"host":"new.example.com"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	v := reload(t, path).VPNs["office"]
	if v.Host != "new.example.com" {
		t.Fatalf("host not applied: %q", v.Host)
	}
	checks := map[string][2]string{
		"version":     {v.Version, "2.6"},
		"tunnel_name": {v.TunnelName, "office-tun"},
		"config":      {v.ConfigFile, "/etc/kongtrol/office.ovpn"},
		"server":      {v.Server, "US"},
		"protocol":    {v.Protocol, "udp"},
		"binary_path": {v.BinaryPath, "/opt/openvpn/sbin/openvpn"},
		"auth.method": {v.Auth.Method, "certificate"},
		"auth.cert":   {v.Auth.Cert, "/etc/kongtrol/office.crt"},
		"auth.key":    {v.Auth.Key, "/etc/kongtrol/office.key"},
	}
	for field, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", field, c[0], c[1])
		}
	}
	if !v.AllowInsecureCert || v.Port != 1194 || v.Priority != 20 {
		t.Errorf("allow_insecure_cert/port/priority lost: %+v", v)
	}
	if v.KillSwitch == nil || *v.KillSwitch {
		t.Errorf("kill_switch override lost: %v", v.KillSwitch)
	}
}

// TestUpdateVPN_DashboardDraftKeepsYAMLOnlyFields mirrors what studio.js
// sends: the full form draft (cert/key now come back from the list) but none
// of the YAML-only fields.
func TestUpdateVPN_DashboardDraftKeepsYAMLOnlyFields(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	body := `{"name":"office","type":"openvpn","priority":30,"host":"vpn.example.com",
		"config":"/etc/kongtrol/office.ovpn","server":"","protocol":"","auth_method":"certificate",
		"cert":"/etc/kongtrol/office.crt","key":"/etc/kongtrol/office.key","username":"","password":""}`
	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleUpdateVPN, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	v := reload(t, path).VPNs["office"]
	if v.Priority != 30 || v.Server != "" {
		t.Errorf("sent fields not applied: priority=%d server=%q", v.Priority, v.Server)
	}
	if v.BinaryPath != "/opt/openvpn/sbin/openvpn" || v.TunnelName != "office-tun" || v.Version != "2.6" || !v.AllowInsecureCert {
		t.Errorf("yaml-only fields lost: %+v", v)
	}
	if v.Auth.Cert != "/etc/kongtrol/office.crt" || v.Auth.Key != "/etc/kongtrol/office.key" {
		t.Errorf("cert/key lost: %+v", v.Auth)
	}
}

func TestListVPNs_ReturnsCertKeyPaths(t *testing.T) {
	s, _ := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodGet, "/api/v1/vpns", "/api/v1/vpns", s.handleListVPNs, "")
	body := rec.Body.String()
	if !strings.Contains(body, `"cert":"/etc/kongtrol/office.crt"`) || !strings.Contains(body, `"key":"/etc/kongtrol/office.key"`) {
		t.Fatalf("cert/key paths missing from list: %s", body)
	}
}

func TestUpdatePolicy_PartialBodyKeepsMatchers(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/policies/corp", "/api/v1/policies/{name}", s.handleUpdatePolicy,
		`{"via":"backup"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	p := reload(t, path).Policies[0]
	if p.Name != "corp" || p.Via != "backup" {
		t.Fatalf("name/via wrong: %+v", p)
	}
	if len(p.Match.IPRanges) != 1 || len(p.Match.Domains) != 1 || len(p.Match.Apps) != 1 {
		t.Fatalf("matchers lost: %+v", p.Match)
	}
}

// TestCreatePolicy_DecodesIPRanges guards the JSON key "ip_ranges": decoding
// straight into config.PolicyRule (yaml tags only) used to drop it.
func TestCreatePolicy_DecodesIPRanges(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPost, "/api/v1/policies", "/api/v1/policies", s.handleCreatePolicy,
		`{"name":"lab","via":"office","match":{"ip_ranges":["192.168.50.0/24"],"domains":[],"apps":[]}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	cfg := reload(t, path)
	if got := cfg.Policies[len(cfg.Policies)-1].Match.IPRanges; len(got) != 1 || got[0] != "192.168.50.0/24" {
		t.Fatalf("ip_ranges not persisted: %v", got)
	}
}

func TestUpdateGroup_OmittedProfilesKept(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/groups/work", "/api/v1/groups/{name}", s.handleUpdateGroup, `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got := reload(t, path).Groups["work"].Profiles; len(got) != 2 {
		t.Fatalf("profiles lost: %v", got)
	}
}

func TestUpdateScheduleRule_PartialBodyKeepsFields(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/scheduler/rules/weekdays", "/api/v1/scheduler/rules/{name}",
		s.handleUpdateScheduleRule, `{"end":"17:00"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	r := reload(t, path).Monitor.Scheduler.Rules[0]
	if r.Name != "weekdays" || r.End != "17:00" || r.Start != "08:00" || len(r.Profiles) != 1 || len(r.Weekdays) != 2 {
		t.Fatalf("rule fields lost: %+v", r)
	}
}

func TestUpdateSettings_PartialBodyKeepsFields(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/settings", "/api/v1/settings", s.handleUpdateSettings,
		`{"scheduler_enabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	cfg := reload(t, path)
	if !cfg.Monitor.Scheduler.Enabled {
		t.Error("scheduler_enabled not applied")
	}
	if cfg.Security.AuditLog.Path != "/var/log/kongtrol/audit.log" || !cfg.Security.AuditLog.Sign {
		t.Errorf("unsent settings lost: %+v", cfg.Security)
	}
}

// emptyYAMLValue matches a key written with an empty scalar/collection value.
var emptyYAMLValue = regexp.MustCompile(`(?m):\s*(""|''|\[\]|\{\}|null|~)\s*$`)

// TestCreateVPN_MinimalWritesNoEmptyKeys creates a minimal profile through
// the API and asserts kongtrol.yaml contains no empty keys for it.
func TestCreateVPN_MinimalWritesNoEmptyKeys(t *testing.T) {
	s, path := newReloadTestServer(t, baseTestYAML, nil)
	rec := doJSON(t, s, http.MethodPost, "/api/v1/vpns", "/api/v1/vpns", s.handleCreateVPN,
		`{"name":"home","type":"wireguard","config":"/etc/wg/home.conf","auth_method":"certificate","host":"","server":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if m := emptyYAMLValue.FindAllString(out, -1); len(m) > 0 {
		t.Fatalf("empty values written: %v\n%s", m, out)
	}
	for _, key := range []string{"host:", "server:", "protocol:", "binary_path:", "tunnel_name:", "version:",
		"username_keychain:", "password_keychain:", "allow_insecure_cert:", "priority:", "port: 0", "kill_switch: null",
		"groups:", "policies:"} {
		if strings.Contains(out, key) {
			t.Errorf("unexpected key %q in:\n%s", key, out)
		}
	}
	home := reload(t, path).VPNs["home"]
	if home.Type != "wireguard" || home.ConfigFile != "/etc/wg/home.conf" {
		t.Fatalf("profile not persisted: %+v", home)
	}
}
