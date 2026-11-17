package forticlient

import (
	"strings"
	"testing"
)

func TestScriptErrorLine(t *testing.T) {
	tests := []struct {
		name, out, want string
	}{
		{
			name: "parse error keeps the message and line, drops the source excerpt",
			out: "At C:/Temp/kongtrol-fcgui-1/fc_connect.ps1:249 char:58\r\n" +
				"+ $password  = 'hunter2' \"x\r\n" +
				"+                                                          ~~~~~\r\n" +
				"Unexpected token 'still' in expression or statement.\r\n" +
				"    + CategoryInfo          : ParserError: (:) [], ParseException\r\n" +
				"    + FullyQualifiedErrorId : UnexpectedToken\r\n",
			want: "Unexpected token 'still' in expression or statement. (script line 249)",
		},
		{
			name: "runtime error: message first",
			out: "Exception calling \"SetText\" with \"1\" argument(s): \"Value cannot be null.\"\r\n" +
				"At C:/Temp/fc_connect.ps1:301 char:5\r\n+ [System.Windows.Forms.Clipboard]::SetText($password)\r\n",
			want: "Exception calling \"SetText\" with \"1\" argument(s): \"Value cannot be null.\" (script line 301)",
		},
		{name: "no output", out: "\r\n", want: "no output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scriptErrorLine(tt.out)
			if got != tt.want {
				t.Fatalf("got %q\nwant %q", got, tt.want)
			}
			if strings.Contains(got, "hunter2") {
				t.Fatal("source excerpt (may hold credentials) leaked into the error")
			}
		})
	}
}

func TestScriptReportedError(t *testing.T) {
	out := "Window 800x600 at (0,0)\r\nERROR: FortiClient connection failed (Access denied)\r\n"
	got, ok := scriptReportedError(out)
	if !ok || got != "FortiClient connection failed (Access denied)" {
		t.Fatalf("got %q, %v", got, ok)
	}
	if _, ok := scriptReportedError("OK: connect sequence complete"); ok {
		t.Fatal("no ERROR line must report none")
	}
}
