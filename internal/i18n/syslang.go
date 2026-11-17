package i18n

import "strings"

// Source says where a resolved display language came from, so clients can
// tell the user ("del sistema" vs a saved choice).
type Source string

const (
	SourceSaved  Source = "saved"  // ~/.kongtrol/preferences.json (kongtrol config lang, dashboard toggle)
	SourceEnv    Source = "env"    // KONGTROL_LANG
	SourceSystem Source = "system" // OS language
)

// FromLocale maps an OS locale or language tag ("es_MX.UTF-8", "es-419",
// "en-US", "fr") to a supported language: es* -> ES, anything else -> EN.
// ok is false for an empty or neutral locale ("C", "POSIX") that says
// nothing about the user's language.
func FromLocale(tag string) (Lang, bool) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	// Strip a codeset/modifier: "es_ES.UTF-8@euro" -> "es_es".
	if i := strings.IndexAny(tag, ".@"); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" || tag == "c" || tag == "posix" {
		return "", false
	}
	if tag == "es" || strings.HasPrefix(tag, "es_") || strings.HasPrefix(tag, "es-") {
		return ES, true
	}
	return EN, true
}

// System returns the OS user's display language (es* -> ES, else EN). When
// the OS gives no usable locale it falls back to Spanish, the default.
func System() Lang {
	if lang, ok := FromLocale(systemLocale()); ok {
		return lang
	}
	return ES
}

// Name is the language's own name, for user-facing lines ("español").
func Name(lang Lang) string {
	if lang == EN {
		return "English"
	}
	return "español"
}

// Other is the language a user would switch to.
func Other(lang Lang) Lang {
	if lang == EN {
		return ES
	}
	return EN
}
