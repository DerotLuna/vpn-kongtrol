package cloudflarewarp

import (
	"errors"
	"strings"
	"testing"
)

func TestConnectError(t *testing.T) {
	base := errors.New("exit status 1")
	for _, tc := range []struct {
		name, out string
		want      []string
		reject    []string
	}{
		{"registration missing", "Error: Registration Missing", []string{"'warp-cli registration new'", "not registered"}, nil},
		{"not registered", "Device not registered", []string{"'warp-cli registration new'"}, nil},
		{"other failure", "daemon unreachable", []string{"warp-cli connect", "daemon unreachable"}, []string{"registration new"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := connectError(tc.out, base).Error()
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("connectError(%q) = %q, want it to contain %q", tc.out, got, w)
				}
			}
			for _, r := range tc.reject {
				if strings.Contains(got, r) {
					t.Errorf("connectError(%q) = %q, must not contain %q", tc.out, got, r)
				}
			}
		})
	}
}
