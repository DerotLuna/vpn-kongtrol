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

// The generated script must parse in Windows PowerShell 5.1 exactly as
// `powershell -File` reads it. Regression: written without a BOM, the
// file was read as Windows-1252 and the UTF-8 em dashes in its strings
// decoded to a closing smart quote, so every run failed with a parse
// error ("connect script: exit status 1 (output: At ...").
// Only the parser runs; the script itself is never executed.
func TestGuiScript_ParsesInWindowsPowerShell(t *testing.T) {
	ps, err := exec.LookPath("powershell")
	if err != nil {
		t.Skip("Windows PowerShell not available")
	}
	for _, action := range []string{"connect", "disconnect"} {
		path := filepath.Join(t.TempDir(), "fc_"+action+".ps1")
		script := buildGuiScript(action, "Office VPN", "josé", "pä'ss—wörd", false)
		if err := writeScriptFile(path, script); err != nil {
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
			t.Fatalf("%s script has parse errors:\n%s", action, out)
		}
	}
}
