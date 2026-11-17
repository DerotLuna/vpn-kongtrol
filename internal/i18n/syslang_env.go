//go:build !windows

package i18n

import "os"

// envLocale applies POSIX precedence for message language: LC_ALL, then
// LC_MESSAGES, then LANG. Neutral values ("C", "POSIX") are skipped so a
// later variable can still decide.
func envLocale() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(name); v != "" {
			if _, ok := FromLocale(v); ok {
				return v
			}
		}
	}
	return ""
}
