//go:build windows

package forticlient

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// windowsPowerShell returns powershell.exe (5.1, what the adapter runs).
// In CI (GitHub sets CI=true) a missing PowerShell fails the test instead
// of skipping it: script parse errors are code errors.
func windowsPowerShell(t *testing.T) string {
	t.Helper()
	ps, err := exec.LookPath("powershell")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("Windows PowerShell not available in CI")
		}
		t.Skip("Windows PowerShell not available")
	}
	return ps
}

// Every generated script must parse in Windows PowerShell 5.1 exactly as
// `powershell -File` reads it. Regression: written without a BOM, the
// file was read as Windows-1252 and the UTF-8 em dashes in its strings
// decoded to a closing smart quote, so every run failed with a parse
// error ("connect script: exit status 1 (output: At ...").
// Only the parser runs; the scripts themselves are never executed.
func TestGuiScript_ParsesInWindowsPowerShell(t *testing.T) {
	ps := windowsPowerShell(t)
	for _, c := range scriptCases {
		path := filepath.Join(t.TempDir(), c.golden)
		if err := writeScriptFile(path, buildGuiScript(c.action, c.insecure)); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(path)
		if !bytes.HasPrefix(raw, []byte(utf8BOM)) {
			t.Fatal("script written without a UTF-8 BOM")
		}
		check := `$t=$null;$e=$null;[void][System.Management.Automation.Language.Parser]::ParseFile($env:KONGTROL_PARSE_SCRIPT,[ref]$t,[ref]$e);` +
			`$e | ForEach-Object { "line $($_.Extent.StartLineNumber): $($_.Message)" }; "errors=$($e.Count)"`
		cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", check)
		cmd.Env = append(os.Environ(), "KONGTROL_PARSE_SCRIPT="+path)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("parser run failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "errors=0") {
			t.Fatalf("%s has parse errors:\n%s", c.golden, out)
		}
	}
}

// The scripts' Read-Input decodes what scriptInputs encodes, including
// quotes PowerShell treats as string delimiters and non-ASCII text. Only
// the Read-Input function runs (no window is touched).
func TestGuiScript_ReadInputRoundTrip(t *testing.T) {
	ps := windowsPowerShell(t)
	tmpl := buildGuiScript(actionConnect, false)
	start := strings.Index(tmpl, "function Read-Input {")
	end := strings.Index(tmpl[start:], "\n}\n")
	if start < 0 || end < 0 {
		t.Fatal("Read-Input not found in the script")
	}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")
	script := tmpl[start:start+end+3] +
		"$a = Read-Input; $b = Read-Input; $c = Read-Input; $d = Read-Input\n" +
		"[IO.File]::WriteAllText($env:KONGTROL_OUT, ($a, $b, $c, $d -join \"`n\"), (New-Object Text.UTF8Encoding $false))\n"
	path := filepath.Join(dir, "read.ps1")
	if err := writeScriptFile(path, script); err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\Program Files\Fortinet\FortiClient\FortiClient.exe`, "Office ’VPN'", "josé", "pä'ss—wörd‘$(x)"}
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path)
	cmd.Env = append(os.Environ(), "KONGTROL_OUT="+outFile)
	cmd.Stdin = strings.NewReader(scriptInputs(want[0], want[1], want[2], want[3]))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != strings.Join(want, "\n") {
		t.Errorf("round trip = %q, want %q", got, strings.Join(want, "\n"))
	}
}
