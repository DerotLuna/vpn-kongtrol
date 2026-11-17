package forticlient

import (
	"regexp"
	"strings"
)

// utf8BOM marks a script file as UTF-8 for Windows PowerShell 5.1.
const utf8BOM = "\xEF\xBB\xBF"

// maxScriptErrorLen bounds the error line kept from a script's output.
const maxScriptErrorLen = 200

// psLocation matches PowerShell's error location line:
// "At C:\...\fc_connect.ps1:147 char:83".
var psLocation = regexp.MustCompile(`^At .*:(\d+) char:\d+`)

// scriptReportedError returns the text of the script's own "ERROR: ..."
// line, if any.
func scriptReportedError(out string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ERROR:") {
			return clip(strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))), true
		}
	}
	return "", false
}

// scriptErrorLine keeps the useful line of a PowerShell error record: the
// message ("Unexpected token ...", "Exception calling ..."), plus the
// script line number from the "At <file>:<line> char:<n>" line. The
// source excerpt lines ("+ ...") are always dropped: they can quote the
// script's own text, which holds the credentials.
func scriptErrorLine(out string) string {
	var msg, lineNo string
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "+") || strings.Trim(line, "~ ") == "" {
			continue
		}
		if m := psLocation.FindStringSubmatch(line); m != nil {
			if lineNo == "" {
				lineNo = m[1]
			}
			continue
		}
		if msg == "" {
			msg = line
		}
		if msg != "" && lineNo != "" {
			break
		}
	}
	if msg == "" {
		return "no output"
	}
	if lineNo != "" {
		msg += " (script line " + lineNo + ")"
	}
	return clip(msg)
}

func clip(s string) string {
	if r := []rune(s); len(r) > maxScriptErrorLen {
		return string(r[:maxScriptErrorLen-1]) + "…"
	}
	return s
}
