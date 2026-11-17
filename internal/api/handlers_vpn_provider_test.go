package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const providerTestYAML = `
vpns:
  nord:
    type: nordvpn
    auto_connect: false
    country: Germany
    group: P2P
    wait_timeout: 10m
    interface: NordLynx
  shark:
    type: surfshark
`

func TestUpdateVPN_ProviderFieldsMerge(t *testing.T) {
	s, path := newReloadTestServer(t, providerTestYAML, nil)
	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/nord", "/api/v1/vpns/{name}", s.handleUpdateVPN,
		`{"city":"Berlin"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	v := reload(t, path).VPNs["nord"]
	if v.City != "Berlin" || v.Country != "Germany" || v.Group != "P2P" || v.WaitTimeout != "10m" || v.Interface != "NordLynx" {
		t.Errorf("provider fields not merged: %+v", v)
	}
	if v.AutoConnect == nil || *v.AutoConnect {
		t.Errorf("auto_connect false lost: %v", v.AutoConnect)
	}

	// auto_connect true is the default: stored as unset.
	rec = doJSON(t, s, http.MethodPut, "/api/v1/vpns/nord", "/api/v1/vpns/{name}", s.handleUpdateVPN,
		`{"auto_connect":true,"group":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	v = reload(t, path).VPNs["nord"]
	if v.AutoConnect != nil || v.Group != "" || v.City != "Berlin" {
		t.Errorf("auto_connect=%v group=%q city=%q", v.AutoConnect, v.Group, v.City)
	}
}

func TestCreateVPN_ProviderWithoutAuth(t *testing.T) {
	s, path := newReloadTestServer(t, providerTestYAML, nil)
	rec := doJSON(t, s, http.MethodPost, "/api/v1/vpns", "/api/v1/vpns", s.handleCreateVPN,
		`{"name":"proton","type":"protonvpn","country":"CH"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	v := reload(t, path).VPNs["proton"]
	if v.Country != "CH" || v.Auth.Method != "credentials" {
		t.Errorf("created profile = %+v", v)
	}
}

func TestUpdateVPN_ValidationErrorLocalized(t *testing.T) {
	s, _ := newReloadTestServer(t, providerTestYAML, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/vpns/{name}", s.handleUpdateVPN)
	for _, c := range []struct{ lang, want string }{
		{"es", "no puede elegir"},
		{"en", "cannot select"},
	} {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/vpns/shark", bytes.NewBufferString(`{"country":"US"}`))
		req.Header.Set("X-Kongtrol-Lang", c.lang)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d: %s", c.lang, rec.Code, rec.Body)
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["error_key"] != "config.err.no_location" || !strings.Contains(body["error"], c.want) || body["hint"] == "" {
			t.Errorf("%s: body = %v", c.lang, body)
		}
	}
}
