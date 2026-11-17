package i18n

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// verb matches fmt verbs (optional explicit index, flags, width,
// precision); %% is a literal percent.
var verb = regexp.MustCompile(`%(?:\[(\d+)\])?[-+# 0]*\d*(?:\.\d+)?([a-zA-Z%])`)

// args returns "argN:verb" for every fmt verb in s, resolving explicit
// indexes (%[2]s) the way fmt does, sorted by argument. Translations may
// reorder arguments with explicit indexes; each argument must still be
// formatted with the same verb.
func args(s string) []string {
	var out []string
	next := 1
	for _, m := range verb.FindAllStringSubmatch(s, -1) {
		if m[2] == "%" {
			continue
		}
		n := next
		if m[1] != "" {
			n, _ = strconv.Atoi(m[1])
		}
		next = n + 1
		out = append(out, fmt.Sprintf("arg%02d:%s", n, m[2]))
	}
	slices.Sort(out)
	return out
}

// TestCatalogParity: every language has exactly the same keys, no empty
// text, and formats the same arguments with the same verbs (F formats
// positionally: a missing verb prints %!(EXTRA …) or garbles the line).
func TestCatalogParity(t *testing.T) {
	base := Keys(ES)
	if len(base) == 0 {
		t.Fatal("empty ES catalog")
	}
	for _, lang := range Langs()[1:] {
		for _, k := range base {
			if _, ok := messages[lang][k]; !ok {
				t.Errorf("%s: missing key %q (present in es)", lang, k)
			}
		}
		for _, k := range Keys(lang) {
			if _, ok := messages[ES][k]; !ok {
				t.Errorf("es: missing key %q (present in %s)", k, lang)
			}
		}
	}
	for _, lang := range Langs() {
		for k, v := range messages[lang] {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: %q is empty", lang, k)
			}
		}
	}
	for _, k := range base {
		es := args(messages[ES][k])
		for _, lang := range Langs()[1:] {
			if v, ok := messages[lang][k]; ok && !slices.Equal(args(v), es) {
				t.Errorf("%s: %q formats %v, es formats %v", lang, k, args(v), es)
			}
		}
	}
}
