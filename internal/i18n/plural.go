package i18n

// Plurals (v1.0.0 · 2026-11-19).
//
// A counted message has one key per CLDR plural category, "<key>.one" and
// "<key>.other". Spanish and English both use only these two categories for
// whole numbers, with "one" exactly for 1 (Unicode CLDR language plural
// rules). A new language with more categories (few, many…) adds its rule to
// pluralCategory, not special cases at the call sites.

import "fmt"

// pluralCategory returns the CLDR cardinal category of n in lang.
func pluralCategory(_ Lang, n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

// N returns the message of key for the count n in lang, formatted with n
// followed by args: N(ES, "tray.status_active", 2) → "2 VPN conectadas".
func N(lang Lang, key string, n int, args ...any) string {
	return fmt.Sprintf(T(lang, key+"."+pluralCategory(lang, n)), append([]any{n}, args...)...)
}
