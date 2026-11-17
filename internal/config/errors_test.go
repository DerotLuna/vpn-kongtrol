package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestValidateProvider_ErrorKeys(t *testing.T) {
	f := false
	cases := []struct {
		v   VPNConfig
		key string
	}{
		{VPNConfig{Type: "wireguard", AutoConnect: &f}, "config.err.provider_field"},
		{VPNConfig{Type: "surfshark", WaitTimeout: "2s"}, "config.err.wait_timeout"},
		{VPNConfig{Type: "protonvpn", Group: "P2P"}, "config.err.group_nord"},
		{VPNConfig{Type: "forticlient", Country: "US"}, "config.err.no_location"},
		{VPNConfig{Type: "nordvpn", Server: "us1", City: "NY"}, "config.err.server_combo"},
	}
	for _, c := range cases {
		// Wrapped like Load does: the key must survive errors.As.
		err := fmt.Errorf("config: validation failed: %w", validateProvider("p", c.v))
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Key != c.key {
			t.Fatalf("%s: err = %v", c.key, err)
		}
		for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
			msg, hint, ok := LocalizeError(err, lang)
			for _, s := range []string{msg, hint} {
				if !ok || s == "" || strings.Contains(s, "config.err.") || strings.Contains(s, "%!") {
					t.Errorf("%s/%s: msg=%q hint=%q", c.key, lang, msg, hint)
				}
			}
		}
		es, _, _ := LocalizeError(err, i18n.ES)
		en, _, _ := LocalizeError(err, i18n.EN)
		if es == en {
			t.Errorf("%s: ES and EN identical: %q", c.key, es)
		}
	}
	if _, _, ok := LocalizeError(errors.New("plain"), i18n.ES); ok {
		t.Error("plain errors must not localize")
	}
}

func TestWithDefaults_AppProvidersNeedNoAuth(t *testing.T) {
	for _, typ := range []string{"protonvpn", "nordvpn", "surfshark"} {
		if got := (VPNConfig{Type: typ}).WithDefaults().Auth.Method; got != "credentials" {
			t.Errorf("%s: method = %q", typ, got)
		}
	}
	if got := (VPNConfig{Type: "forticlient"}).WithDefaults().Auth.Method; got != "" {
		t.Errorf("forticlient must keep its own auth: %q", got)
	}
}
