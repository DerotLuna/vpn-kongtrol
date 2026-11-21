package config

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// ASCIIPolicyDomain normalizes a DNS name or a one-label wildcard pattern.
// The root dot is preserved for lookups so DNS search suffixes stay disabled.
func ASCIIPolicyDomain(value string) string {
	domain := strings.TrimSpace(value)
	prefix := ""
	if rest, ok := strings.CutPrefix(domain, "*."); ok {
		prefix, domain = "*.", rest
	}
	// Preserve compatibility with ASCII local names such as under_score.
	for i := 0; i < len(domain); i++ {
		if domain[i] >= utf8.RuneSelf {
			ascii, err := idna.Lookup.ToASCII(domain)
			if err != nil {
				return ""
			}
			return prefix + ascii
		}
	}
	return prefix + strings.ToLower(domain)
}
