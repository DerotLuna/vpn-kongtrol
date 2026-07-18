package main

import (
	"os"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func cliLang() i18n.Lang {
	if lang := strings.ToLower(strings.TrimSpace(os.Getenv("KONGTROL_LANG"))); lang != "" {
		if strings.HasPrefix(lang, "es") {
			return i18n.ES
		}
		if strings.HasPrefix(lang, "en") {
			return i18n.EN
		}
	}
	if p, err := loadPreferences(); err == nil {
		switch strings.ToLower(strings.TrimSpace(p.Language)) {
		case "es":
			return i18n.ES
		case "en":
			return i18n.EN
		}
	}
	// No env override and no saved preference: default to English.
	return i18n.EN
}

func ct(key string) string {
	return i18n.T(cliLang(), key)
}

func cf(key string, args ...any) string {
	return i18n.F(cliLang(), key, args...)
}
