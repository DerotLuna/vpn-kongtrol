package main

import "testing"

func TestParseOpenRequest(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"--open", "/profiles"}, "/profiles"},
		{[]string{"--open=/profiles"}, "/profiles"},
		{[]string{"--open", "/"}, "/"},
		{[]string{"--open"}, ""},
		{[]string{"--open", "https://evil.example"}, ""},
		{[]string{"--open", "//evil.example"}, ""},
		{[]string{"--open=/api/v1/shutdown"}, ""},
		{[]string{"-psn_0_123"}, ""},
	}
	for _, c := range cases {
		if got := parseOpenRequest(c.args); got != c.want {
			t.Errorf("parseOpenRequest(%q)=%q, want %q", c.args, got, c.want)
		}
	}
}

func TestTakeOpenRequestOnce(t *testing.T) {
	setOpenRequest("/profiles")
	if got := takeOpenRequest(); got != "/profiles" {
		t.Fatalf("first take=%q", got)
	}
	if got := takeOpenRequest(); got != "" {
		t.Fatalf("second take=%q, want empty", got)
	}
}
