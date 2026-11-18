package api

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

type nopDNSGuard struct{ on bool }

func (g *nopDNSGuard) Apply(string, []net.IP) error { g.on = true; return nil }
func (g *nopDNSGuard) Restore() error               { g.on = false; return nil }
func (g *nopDNSGuard) IsActive() bool               { return g.on }

// security/status carries the DNS guard servers in use and a translated
// kill switch error; security/blocked lists the prefixes in place.
func TestSecurityStatus_DNSServersKillSwitchErrorAndBlocked(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.dnsMgr = monitor.NewDNSManager(&nopDNSGuard{}, zap.NewNop())
	s.dnsMgr.OnConnect("office", "tun0", []net.IP{net.ParseIP("10.0.0.53")})
	st := security.KillSwitchStatus{State: security.KillSwitchActive, Blocking: []string{"office"},
		BlockedCount: 2, Error: "helper not installed", ErrorKey: "service.error.not_installed"}
	s.SetKillSwitchStatus(func() security.KillSwitchStatus { return st })
	s.SetBlockedList(func() []string { return []string{"10.20.0.0/16", "160.79.104.10/32"} })

	rec := doJSONLang(t, s, http.MethodGet, "/api/v1/security/status", "/api/v1/security/status", s.handleSecurityStatus, "", "es")
	var body struct {
		DNSGuardServers  []string            `json:"dns_guard_servers"`
		DNSGuardProfiles map[string][]string `json:"dns_guard_profiles"`
		KillSwitchError  *killSwitchErrorDTO `json:"kill_switch_error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.DNSGuardServers) != 1 || body.DNSGuardServers[0] != "10.0.0.53" || len(body.DNSGuardProfiles["office"]) != 1 {
		t.Fatalf("dns guard = %+v", body)
	}
	e := body.KillSwitchError
	if e == nil || e.ErrorKey != "service.error.not_installed" || e.Message == "" || e.Message == e.ErrorKey || e.Detail == "" {
		t.Fatalf("kill_switch_error = %+v", e)
	}

	// An error with no guidance key gets the generic line + what to do.
	st = security.KillSwitchStatus{State: security.KillSwitchArmed, Error: "boom"}
	if got := localizeKillSwitchError(st, "en"); got == nil || got.ErrorKey != "killswitch.err.apply" || got.Hint == "" {
		t.Fatalf("generic = %+v", got)
	}
	if localizeKillSwitchError(security.KillSwitchStatus{}, "en") != nil {
		t.Fatal("no error must give nil")
	}

	rec = doJSON(t, s, http.MethodGet, "/api/v1/security/blocked", "/api/v1/security/blocked", s.handleSecurityBlocked, "")
	var blocked struct {
		State    string   `json:"state"`
		Prefixes []string `json:"prefixes"`
		Count    int      `json:"count"`
		Blocking []string `json:"blocking"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &blocked); err != nil {
		t.Fatal(err)
	}
	if blocked.Count != 2 || len(blocked.Prefixes) != 2 || blocked.State != "armed" {
		t.Fatalf("blocked = %+v", blocked)
	}
}

// expected_ips: read as an object, replaced whole, validated (existing
// profile, one IP) with translated errors.
func TestSettings_ExpectedIPs(t *testing.T) {
	s, path := newReloadTestServer(t, mergeTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/settings", "/api/v1/settings", s.handleUpdateSettings,
		`{"expected_ips":{"office":" 203.0.113.7 ","backup":""}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	got := reload(t, path).Security.IntegrityCheck.ExpectedIPs
	if len(got) != 1 || got["office"] != "203.0.113.7" {
		t.Fatalf("stored = %v", got)
	}
	var dto settingsDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &dto)
	if dto.ExpectedIPs["office"] != "203.0.113.7" {
		t.Fatalf("dto = %v", dto.ExpectedIPs)
	}
	for body, key := range map[string]string{
		`{"expected_ips":{"ghost":"1.2.3.4"}}`:     "config.err.expected_ip_profile",
		`{"expected_ips":{"office":"1.2.3.4/32"}}`: "config.err.expected_ip",
	} {
		rec := doJSONLang(t, s, http.MethodPut, "/api/v1/settings", "/api/v1/settings", s.handleUpdateSettings, body, "es")
		if rec.Code != http.StatusBadRequest || decodeLocalized(t, rec).ErrorKey != key {
			t.Fatalf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	if got := reload(t, path).Security.IntegrityCheck.ExpectedIPs; got["office"] != "203.0.113.7" {
		t.Fatalf("rejected update changed the file: %v", got)
	}
}
