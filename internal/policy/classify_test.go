package policy

import (
	"reflect"
	"testing"
)

func TestClassifyTargets(t *testing.T) {
	cases := []struct {
		name       string
		in         []string
		subdomains bool
		want       Targets
		invalid    []string
	}{
		{"domain with subdomains", []string{"github.com"}, true,
			Targets{Domains: []string{"github.com", "*.github.com"}, First: "github.com"}, nil},
		{"domain without subdomains", []string{"GitHub.com."}, false,
			Targets{Domains: []string{"github.com"}, First: "github.com"}, nil},
		{"url becomes its host", []string{"https://Intranet.Example.com:8443/a?b#c"}, true,
			Targets{Domains: []string{"intranet.example.com", "*.intranet.example.com"}, First: "intranet.example.com"}, nil},
		{"host with port and path", []string{"example.com:443/x"}, false,
			Targets{Domains: []string{"example.com"}, First: "example.com"}, nil},
		{"wildcard kept as written", []string{"*.corp.example"}, true,
			Targets{Domains: []string{"*.corp.example"}, First: "*.corp.example"}, nil},
		{"ip and cidr", []string{"10.0.0.5", "10.0.0.0/8", "2001:db8::/32", "::1"}, true,
			Targets{IPRanges: []string{"10.0.0.5", "10.0.0.0/8", "2001:db8::/32", "::1"}, First: "10.0.0.5"}, nil},
		{"ip inside a url", []string{"http://192.168.1.10:8080/"}, true,
			Targets{IPRanges: []string{"192.168.1.10"}, First: "192.168.1.10"}, nil},
		{"apps", []string{"app:chrome.exe", "APP: C:\\Program Files\\x.exe "}, true,
			Targets{Apps: []string{"chrome.exe", "C:\\Program Files\\x.exe"}, First: "chrome.exe"}, nil},
		{"mixed and deduplicated", []string{"a.com", "a.com", "10.1.0.0/16", "app:x"}, true,
			Targets{Domains: []string{"a.com", "*.a.com"}, IPRanges: []string{"10.1.0.0/16"}, Apps: []string{"x"}, First: "a.com"}, nil},
		{"invalid entries", []string{"not a host", "app:", "10.0.0.0/99", "", "ok.com"}, false,
			Targets{Domains: []string{"ok.com"}, First: "ok.com"}, []string{"not a host", "app:", "10.0.0.0/99"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, invalid := ClassifyTargets(c.in, c.subdomains)
			if !reflect.DeepEqual(got, c.want) || !reflect.DeepEqual(invalid, c.invalid) {
				t.Fatalf("got %+v invalid %q, want %+v invalid %q", got, invalid, c.want, c.invalid)
			}
		})
	}
}

func TestAutoName(t *testing.T) {
	cases := []struct{ first, via, want string }{
		{"github.com", "oficina", "github.com por oficina"},
		{`C:\Program Files\App\x.exe`, "casa", "x.exe por casa"},
		{"/usr/bin/curl", "casa", "curl por casa"},
		{`a"b<c>`, "v", "abc por v"},
	}
	for _, c := range cases {
		if got := AutoName("%[1]s por %[2]s", c.first, c.via, nil); got != c.want {
			t.Errorf("AutoName(%q) = %q, want %q", c.first, got, c.want)
		}
	}
	long := AutoName("%[1]s por %[2]s", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com", "v", nil)
	if n := len([]rune(long)); n > 60 {
		t.Fatalf("long name has %d runes", n)
	}
	if got := AutoName("%[1]s via %[2]s", "a.com", "v", []string{"a.com via v"}); got != "a.com via v 2" {
		t.Fatal(got)
	}
}

func TestSuggestName(t *testing.T) {
	existing := []string{"github.com por oficina", "GITHUB.COM POR OFICINA 2"}
	if got := SuggestName("x por y", existing); got != "x por y" {
		t.Fatal(got)
	}
	if got := SuggestName("github.com por oficina", existing); got != "github.com por oficina 3" {
		t.Fatal(got)
	}
}
