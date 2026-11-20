package main

import "testing"

func TestWantsVersion(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"--version"}, true},
		{[]string{"-version"}, true},
		{[]string{"--open", "/profiles"}, false},
		{[]string{"--open", "/", "--version"}, true},
		{[]string{"version"}, false},
	}
	for _, c := range cases {
		if got := wantsVersion(c.args); got != c.want {
			t.Errorf("wantsVersion(%q)=%v, want %v", c.args, got, c.want)
		}
	}
}

func TestVersionText(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	for in, want := range map[string]string{"": "dev", "v0.7.0": "0.7.0", "0.7.0-rc.4": "0.7.0-rc.4"} {
		version = in
		if got := versionText(); got != want {
			t.Errorf("versionText(%q)=%q, want %q", in, got, want)
		}
	}
}
