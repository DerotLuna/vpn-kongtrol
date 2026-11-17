//go:build darwin

package i18n

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// systemLocale uses the POSIX variables first; GUI-launched apps (the tray)
// usually have none, so it falls back to the first entry of the user's
// AppleLanguages list.
func systemLocale() string {
	if v := envLocale(); v != "" {
		return v
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	return firstAppleLanguage(string(out))
}

// firstAppleLanguage extracts the first tag from `defaults read -g
// AppleLanguages`, a plist array such as:
//
//	(
//	    "es-MX",
//	    "en-US"
//	)
func firstAppleLanguage(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.Trim(strings.TrimSpace(line), `",`)
		if line == "" || line == "(" || line == ")" {
			continue
		}
		return line
	}
	return ""
}
