package config

import (
	"strings"
	"testing"
)

func TestValidateProvider(t *testing.T) {
	f := false
	cases := []struct {
		name    string
		v       VPNConfig
		wantErr string // substring; "" = valid
	}{
		{"existing forticlient", VPNConfig{Type: "forticlient", Host: "vpn.x", Port: 443, TunnelName: "Office", Version: "6.4"}, ""},
		{"forticlient attach fields", VPNConfig{Type: "forticlient", AutoConnect: &f, WaitTimeout: "10m", Interface: "Fortinet*"}, ""},
		{"forticlient location", VPNConfig{Type: "forticlient", Country: "US"}, "cannot select"},
		{"surfshark minimal", VPNConfig{Type: "surfshark"}, ""},
		{"surfshark server", VPNConfig{Type: "surfshark", Server: "us-nyc"}, "cannot select"},
		{"nord legacy server", VPNConfig{Type: "nordvpn", Server: "United States"}, ""},
		{"nord country+city+group", VPNConfig{Type: "nordvpn", Country: "Germany", City: "Berlin", Group: "P2P"}, ""},
		{"nord server+country", VPNConfig{Type: "nordvpn", Server: "us3710", Country: "US"}, "exact server"},
		{"proton group", VPNConfig{Type: "protonvpn", Group: "P2P"}, "only supported by nordvpn"},
		{"proton city", VPNConfig{Type: "protonvpn", City: "Zurich"}, ""},
		{"bad wait_timeout", VPNConfig{Type: "surfshark", WaitTimeout: "soon"}, "wait_timeout"},
		{"short wait_timeout", VPNConfig{Type: "surfshark", WaitTimeout: "2s"}, "wait_timeout"},
		{"attach field on wireguard", VPNConfig{Type: "wireguard", AutoConnect: &f}, "auto_connect only applies"},
		{"location on openvpn", VPNConfig{Type: "openvpn", Country: "US"}, "country only applies"},
	}
	for _, c := range cases {
		err := validateProvider("p", c.v)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: err = %v, want %q", c.name, err, c.wantErr)
		}
	}
}

func TestAdapterExtra(t *testing.T) {
	f := false
	v := VPNConfig{Server: "s", Country: "c", City: "ci", Group: "g", Interface: "i", WaitTimeout: "1m", AppPath: "/a", AutoConnect: &f}
	e := v.AdapterExtra()
	for k, want := range map[string]string{"server": "s", "country": "c", "city": "ci", "group": "g",
		"interface": "i", "wait_timeout": "1m", "app_path": "/a", "auto_connect": "false"} {
		if e[k] != want {
			t.Errorf("Extra[%q] = %q, want %q", k, e[k], want)
		}
	}
	if _, ok := (VPNConfig{}).AdapterExtra()["auto_connect"]; ok {
		t.Error("unset auto_connect must stay absent (adapter default true)")
	}
}
