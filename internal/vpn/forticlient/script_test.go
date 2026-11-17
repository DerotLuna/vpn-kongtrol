package forticlient

import (
	"encoding/base64"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite testdata/*.ps1 golden files")

// scriptCases are the scripts Kongtrol ships; each has a golden file and
// is parsed by Windows PowerShell in gui_windows_test.go.
var scriptCases = []struct {
	golden   string
	action   string
	insecure bool
}{
	{"fc_connect.ps1", actionConnect, false},
	{"fc_connect_insecure.ps1", actionConnect, true},
	{"fc_disconnect.ps1", actionDisconnect, false},
	{"fc_auth.ps1", actionAuth, false},
}

// Golden files make every change to the generated PowerShell visible in
// review. Regenerate with: go test ./internal/vpn/forticlient -run Golden -update
func TestGuiScript_Golden(t *testing.T) {
	for _, c := range scriptCases {
		got := buildGuiScript(c.action, c.insecure)
		path := filepath.Join("testdata", c.golden)
		if *updateGolden {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (run with -update)", path, err)
		}
		if strings.ReplaceAll(string(want), "\r\n", "\n") != got {
			t.Errorf("%s differs from the generated script; review, then run with -update", path)
		}
	}
}

func TestGuiScript_NoPlaceholdersAndPSBool(t *testing.T) {
	for _, c := range scriptCases {
		s := buildGuiScript(c.action, c.insecure)
		if strings.Contains(s, "{{") || strings.Contains(s, "}}") {
			t.Errorf("%s: unreplaced placeholder", c.golden)
		}
		if !strings.Contains(s, "$action            = '"+c.action+"'") {
			t.Errorf("%s: action not set", c.golden)
		}
		// A bare `true` / `false` runs as a command and leaves $null.
		want := "$allowInsecureCert = $false"
		if c.insecure {
			want = "$allowInsecureCert = $true"
		}
		if !strings.Contains(s, want) {
			t.Errorf("%s: want %q", c.golden, want)
		}
	}
}

// The script text is the same whatever the credentials: they only travel
// on stdin, so they can't break quoting or land in the temp file.
func TestScriptInputs(t *testing.T) {
	in := scriptInputs(`C:\Fortinet\FortiClient.exe`, "Office ’VPN'", "josé", "pä'ss—wörd‘")
	lines := strings.Split(strings.TrimSuffix(in, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4", len(lines))
	}
	want := []string{`C:\Fortinet\FortiClient.exe`, "Office ’VPN'", "josé", "pä'ss—wörd‘"}
	for i, l := range lines {
		b, err := base64.StdEncoding.DecodeString(l)
		if err != nil || string(b) != want[i] {
			t.Errorf("line %d = %q (%v), want %q", i, b, err, want[i])
		}
		if strings.Contains(l, "pä") || strings.Contains(l, "josé") {
			t.Errorf("line %d not encoded", i)
		}
	}
}
