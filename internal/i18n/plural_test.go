package i18n

import (
	"strings"
	"testing"
)

func TestNPlural(t *testing.T) {
	cases := []struct {
		lang Lang
		n    int
		want string
	}{
		{ES, 0, "0 perfiles VPN conectados"},
		{ES, 1, "1 perfil VPN conectado"},
		{ES, 2, "2 perfiles VPN conectados"},
		{EN, 0, "0 VPN profiles connected"},
		{EN, 1, "1 VPN profile connected"},
		{EN, 3, "3 VPN profiles connected"},
	}
	for _, c := range cases {
		if got := N(c.lang, "tray.status_active", c.n); got != c.want {
			t.Errorf("N(%s, %d) = %q, want %q", c.lang, c.n, got, c.want)
		}
	}
}

// Every counted key ("<key>.one") also has "<key>.other" in every language
// (".other" alone is not checked: some plain keys end in it, e.g.
// doctor.path.other).
func TestPluralKeysComplete(t *testing.T) {
	for _, lang := range Langs() {
		m := messages[lang]
		for k := range m {
			if base, ok := strings.CutSuffix(k, ".one"); ok {
				if _, ok := m[base+".other"]; !ok {
					t.Errorf("%s: %s has no %s.other", lang, k, base)
				}
			}
		}
	}
}
