package docgen

import (
	"os"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// TestTextParity: doc texts exist in every language, non-empty.
func TestTextParity(t *testing.T) {
	for _, cat := range []map[i18n.Lang]map[string]string{docText, fieldText} {
		for _, lang := range i18n.Langs() {
			for k := range cat[i18n.ES] {
				if strings.TrimSpace(cat[lang][k]) == "" {
					t.Errorf("%s: %q missing or empty", lang, k)
				}
			}
			for k := range cat[lang] {
				if _, ok := cat[i18n.ES][k]; !ok {
					t.Errorf("es: %q missing (present in %s)", k, lang)
				}
			}
		}
	}
}

// TestConfigFieldsDocumented reports schema fields without a description
// (and stale descriptions). Docs never block a merge (AGENTS.md): this only
// fails with KONGTROL_DOCS_STRICT=1; CI reports it as a warning through
// `go run ./cmd/kongtrol gen-docs --check`.
func TestConfigFieldsDocumented(t *testing.T) {
	for _, m := range MissingDescriptions() {
		if os.Getenv("KONGTROL_DOCS_STRICT") == "1" {
			t.Error(m)
		} else {
			t.Log("warning:", m)
		}
	}
}
