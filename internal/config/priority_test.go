package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ADR-0108: priority is 0 (unset: position decides) or 1..MaxPriority.
func TestValidatePriorityRange(t *testing.T) {
	for _, p := range []int{0, 1, 20, MaxPriority} {
		c := &Config{VPNs: map[string]VPNConfig{"casa": {Type: "nordvpn", Auth: AuthConfig{Method: "none"}, Priority: p}}}
		if err := Validate(c); err != nil {
			t.Errorf("priority %d: %v", p, err)
		}
	}
	for _, p := range []int{-1, MaxPriority + 1} {
		c := &Config{VPNs: map[string]VPNConfig{"casa": {Type: "nordvpn", Auth: AuthConfig{Method: "none"}, Priority: p}}}
		var ve *ValidationError
		err := Validate(c)
		if !errors.As(err, &ve) || ve.Key != "config.err.profile_priority" {
			t.Fatalf("priority %d: %v", p, err)
		}
		if hint := ve.Hint(i18n.ES); !strings.Contains(hint, "1000") {
			t.Fatalf("hint = %q", hint)
		}
	}
}
