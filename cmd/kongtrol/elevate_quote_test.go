package main

import "testing"

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"":              "''",
		"plain":         "'plain'",
		"it's":          `'it'\''s'`,
		`a "b" $c \d`:   `'a "b" $c \d'`,
		"/Apps/K T/bin": "'/Apps/K T/bin'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestAppleScriptAdminCommand(t *testing.T) {
	got, err := appleScriptAdminCommand([]string{`/Apps/K "T"/kongtrol`, "service", "--user-name", `o'b\x`})
	if err != nil {
		t.Fatal(err)
	}
	want := `do shell script "'/Apps/K \"T\"/kongtrol' 'service' '--user-name' 'o'\\''b\\x'" with administrator privileges`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if _, err := appleScriptAdminCommand([]string{"a\nb"}); err == nil {
		t.Error("newline accepted")
	}
}
