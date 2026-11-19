package profileimport

import (
	"strings"
	"testing"
)

func TestCleanName(t *testing.T) {
	cases := map[string]string{
		"oficina.conf":                     "oficina",
		`C:\Users\ana\Mi Oficina (2).ovpn`: "Mi-Oficina-2",
		"/tmp/__.vpn..x.conf":              "vpn.x",
		"ñandú.conf":                       "and",
		"   .conf":                         "",
		strings.Repeat("a", 80) + ".conf":  strings.Repeat("a", 64),
	}
	for in, want := range cases {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
		if got := CleanName(in); got != "" && !nameRE.MatchString(got) {
			t.Errorf("CleanName(%q) = %q is not a valid import name", in, got)
		}
	}
}
