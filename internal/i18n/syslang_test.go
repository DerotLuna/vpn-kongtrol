package i18n

import "testing"

func TestFromLocale(t *testing.T) {
	cases := []struct {
		in   string
		want Lang
		ok   bool
	}{
		{"es_MX.UTF-8", ES, true},
		{"es-419", ES, true},
		{"ES", ES, true},
		{"es_ES.UTF-8@euro", ES, true},
		{"en-US", EN, true},
		{"fr_FR.UTF-8", EN, true},
		{"estonian", EN, true}, // not "es" + separator
		{"C", "", false},
		{"POSIX", "", false},
		{"C.UTF-8", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := FromLocale(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("FromLocale(%q) = %q,%v; want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestOtherAndName(t *testing.T) {
	if Other(ES) != EN || Other(EN) != ES {
		t.Fatal("Other must swap es/en")
	}
	if Name(ES) != "español" || Name(EN) != "English" {
		t.Fatal("unexpected language names")
	}
}

func TestSystemResolves(t *testing.T) {
	got := System()
	if got != ES && got != EN {
		t.Fatalf("System() = %q", got)
	}
	t.Logf("system locale %q -> %s", systemLocale(), got)
}
