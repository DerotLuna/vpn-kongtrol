package main

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestProviderFields_MinimalPerProvider(t *testing.T) {
	all := providerAnswers{
		AutoConnect: false, Country: "DE", City: "Berlin", Group: "P2P",
		Interface: "NordLynx", AppPath: "/opt/app", WaitTimeout: "10m",
	}
	cases := []struct {
		typ  string
		a    providerAnswers
		want [][2]string
	}{
		{"protonvpn", defaultProviderAnswers(), nil},
		{"nordvpn", providerAnswers{AutoConnect: true, WaitTimeout: "5m"}, nil}, // defaults never written
		{"surfshark", all, [][2]string{{"interface", "NordLynx"}, {"app_path", "/opt/app"}, {"wait_timeout", "10m"}}},
		{"forticlient", all, [][2]string{{"auto_connect", "false"}, {"interface", "NordLynx"}, {"app_path", "/opt/app"}, {"wait_timeout", "10m"}}},
		{"protonvpn", all, [][2]string{{"auto_connect", "false"}, {"country", "DE"}, {"city", "Berlin"}, {"interface", "NordLynx"}, {"app_path", "/opt/app"}, {"wait_timeout", "10m"}}},
		{"nordvpn", all, [][2]string{{"auto_connect", "false"}, {"country", "DE"}, {"city", "Berlin"}, {"group", "P2P"}, {"interface", "NordLynx"}, {"app_path", "/opt/app"}, {"wait_timeout", "10m"}}},
		{"nordvpn", providerAnswers{AutoConnect: true, Server: " us3710 "}, [][2]string{{"server", "us3710"}}},
	}
	for _, c := range cases {
		got := providerFields(c.typ, c.a)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.typ, got, c.want)
		}
		// The written profile must load and validate as-is (no auth block).
		node := mapNode(append([][2]string{{"type", c.typ}}, got...))
		if c.typ == "forticlient" {
			node.Content = append(node.Content, scalarNode("auth"), mapNode([][2]string{{"method", "credentials"}}))
		}
		raw, err := yaml.Marshal(node)
		if err != nil {
			t.Fatal(err)
		}
		var v config.VPNConfig
		if err := yaml.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		cfg := config.Config{VPNs: map[string]config.VPNConfig{"p": v.WithDefaults()}}
		if err := config.Validate(&cfg); err != nil {
			t.Errorf("%s: written YAML invalid: %v\n%s", c.typ, err, raw)
		}
	}
}

func TestValidateWaitTimeout(t *testing.T) {
	v := validateWaitTimeout("en")
	for in, ok := range map[string]bool{"": true, "5m": true, "10s": true, "2s": false, "soon": false} {
		if (v(in) == nil) != ok {
			t.Errorf("%q: ok=%v", in, !ok)
		}
	}
}

func TestNextStepsProfile(t *testing.T) {
	if got := nextStepsProfile("es", []profileSummary{{name: "casa"}}, nil); got != "casa" {
		t.Errorf("added = %q", got)
	}
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{"zeta": {}, "alfa": {}}}
	if got := nextStepsProfile("es", nil, cfg); got != "alfa" {
		t.Errorf("configured = %q", got)
	}
	if got := nextStepsProfile("es", nil, nil); got != "<perfil>" {
		t.Errorf("placeholder = %q", got)
	}
}
