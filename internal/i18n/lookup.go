package i18n

import "sort"

// Langs lists every supported language, Spanish (the default) first.
func Langs() []Lang { return []Lang{ES, EN} }

// Keys returns the sorted keys of lang's catalog (for parity tests and
// generators).
func Keys(lang Lang) []string {
	keys := make([]string, 0, len(messages[lang]))
	for k := range messages[lang] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// KeysFor returns, sorted, every key whose text equals value in any
// language. Used to re-translate strings that were rendered once at
// start-up (e.g. cobra flag usages set in init()) into another language.
func KeysFor(value string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, m := range messages {
		for k, v := range m {
			if v == value && !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}
