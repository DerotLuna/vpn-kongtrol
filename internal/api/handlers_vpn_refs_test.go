package api

import (
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const refsTestYAML = `
vpns:
  office:
    type: openvpn
    failover: [backup]
    auth:
      method: credentials
      username_keychain: office.username
      password_keychain: office.password
  backup:
    type: wireguard
    auth:
      method: certificate
  spare:
    failover: [office]
    type: wireguard
    auth:
      method: certificate
groups:
  work:
    profiles: [office, spare]
policies:
  - name: corp
    via: office
    match:
      domains: ["*.corp.example"]
security:
  integrity_check:
    expected_ips:
      office: 203.0.113.7
monitor:
  scheduler:
    rules:
      - name: weekdays
        profiles: [office]
`

// Renaming moves every reference, the expected IP and the keychain
// entries, and keeps the rest of the profile.
func TestUpdateVPN_RenameMovesEverything(t *testing.T) {
	keyring.MockInit()
	s, path := newReloadTestServer(t, refsTestYAML, nil)
	s.preferencesPath = filepath.Join(t.TempDir(), "preferences.json")
	_ = config.SavePreferences(s.preferencesPath, &config.Preferences{Favorites: []string{"office"}})
	_ = config.SetCredential("office", "username", "ana")
	_ = config.SetCredential("office", "password", "s3cret")

	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleUpdateVPN, `{"name":"oficina"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"renamed"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	c := reload(t, path)
	if _, ok := c.VPNs["office"]; ok {
		t.Fatal("old name still there")
	}
	v := c.VPNs["oficina"]
	if v.Type != "openvpn" || !slices.Equal(v.Failover, []string{"backup"}) ||
		v.Auth.UsernameKeychain != "oficina.username" || v.Auth.PasswordKeychain != "oficina.password" {
		t.Fatalf("renamed profile = %+v", v)
	}
	if c.Policies[0].Via != "oficina" || !slices.Equal(c.Groups["work"].Profiles, []string{"oficina", "spare"}) ||
		!slices.Equal(c.VPNs["spare"].Failover, []string{"oficina"}) ||
		!slices.Equal(c.Monitor.Scheduler.Rules[0].Profiles, []string{"oficina"}) ||
		c.Security.IntegrityCheck.ExpectedIPs["oficina"] != "203.0.113.7" {
		t.Fatalf("references not moved: %+v", c)
	}
	if pw, err := config.GetCredential("oficina", "password"); err != nil || pw != "s3cret" {
		t.Fatalf("password not moved: %q %v", pw, err)
	}
	if _, err := config.GetCredential("office", "password"); err == nil {
		t.Fatal("old password left behind")
	}
	if p, _ := config.LoadPreferences(s.preferencesPath); !slices.Equal(p.Favorites, []string{"oficina"}) {
		t.Fatalf("favorites = %v", p.Favorites)
	}
}

// A rename onto an existing name, an invalid name or a tunnel that is up
// is refused with a translated error and changes nothing.
func TestUpdateVPN_RenameRefusals(t *testing.T) {
	keyring.MockInit()
	cases := map[string]struct {
		body   string
		up     bool
		status int
		key    string
	}{
		"exists":    {`{"name":"spare"}`, false, http.StatusConflict, "api.err.vpn_exists"},
		"bad name":  {`{"name":"a<b"}`, false, http.StatusBadRequest, "config.err.name_chars"},
		"connected": {`{"name":"x"}`, true, http.StatusConflict, "api.err.vpn_rename_busy"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			st := vpn.StatusDisconnected
			if c.up {
				st = vpn.StatusConnected
			}
			s, path := newReloadTestServer(t, refsTestYAML, map[string]vpn.VPNAdapter{"office": newWSTestAdapter(st)})
			rec := doJSONLang(t, s, http.MethodPut, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleUpdateVPN, c.body, "es")
			if rec.Code != c.status || decodeLocalized(t, rec).ErrorKey != c.key {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			if _, ok := reload(t, path).VPNs["office"]; !ok {
				t.Fatal("config changed")
			}
		})
	}
}

// Delete lists every reference at once; an unreferenced profile is
// deleted and its expected IP goes with it.
func TestDeleteVPN_References(t *testing.T) {
	keyring.MockInit()
	s, path := newReloadTestServer(t, refsTestYAML, nil)
	rec := doJSONLang(t, s, http.MethodDelete, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleDeleteVPN, "", "en")
	e := decodeLocalized(t, rec)
	if rec.Code != http.StatusConflict || e.ErrorKey != "api.err.vpn_in_use" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	for _, want := range []string{"corp", "work", "backup", "weekdays"} {
		if !strings.Contains(e.Error, want) {
			t.Errorf("message %q does not name %s", e.Error, want)
		}
	}
	// backup is only another profile's failover target.
	rec = doJSON(t, s, http.MethodDelete, "/api/v1/vpns/backup", "/api/v1/vpns/{name}", s.handleDeleteVPN, "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "office") {
		t.Fatalf("failover reference not checked: %d %s", rec.Code, rec.Body)
	}

	// Free office of everything, then it can go (expected IP dropped).
	c := reload(t, path)
	c.Policies = nil
	c.Groups = nil
	c.Monitor.Scheduler.Rules = nil
	b := c.VPNs["spare"]
	b.Failover = nil
	c.VPNs["spare"] = b
	if err := s.saveRuntimeConfig(path, c); err != nil {
		t.Fatal(err)
	}
	rec = doJSON(t, s, http.MethodDelete, "/api/v1/vpns/office", "/api/v1/vpns/{name}", s.handleDeleteVPN, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	c = reload(t, path)
	if _, ok := c.VPNs["office"]; ok || c.Security.IntegrityCheck.ExpectedIPs["office"] != "" {
		t.Fatalf("not deleted cleanly: %+v", c.Security.IntegrityCheck)
	}
}
