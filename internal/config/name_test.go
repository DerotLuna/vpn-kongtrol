package config

import (
	"errors"
	"strings"
	"testing"
)

// Regression: profile names reached the legacy dashboard's innerHTML and
// inline onclick unescaped; a name like x');fetch('/api/v1/data/purge… was
// accepted by config validation (stored XSS in the local panel).
func TestValidateName(t *testing.T) {
	ok := []string{"oficina", "Oficina VPN", "casa-2", "proton_es.1", "trabajo ñandú"}
	bad := []string{
		"", "   ", "x');alert(1)//", `a"b`, "a<b", "a>b", "a&b", "a`b", `a\b`,
		"a\nb", "a\x00b", "a b", strings.Repeat("a", 65),
	}
	for _, n := range ok {
		if err := ValidateName("vpn", n); err != nil {
			t.Errorf("%q rejected: %v", n, err)
		}
	}
	for _, n := range bad {
		err := ValidateName("vpn", n)
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Key != "config.err.name_chars" {
			t.Errorf("%q accepted or wrong error: %v", n, err)
		}
	}
}

func TestValidate_RejectsUnsafeProfileName(t *testing.T) {
	cfg := &Config{VPNs: map[string]VPNConfig{`x');alert(1)//`: {Type: "wireguard", ConfigFile: "/tmp/x.conf"}}}
	err := validateSemantics(cfg)
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Key != "config.err.name_chars" {
		t.Fatalf("err = %v", err)
	}
}
