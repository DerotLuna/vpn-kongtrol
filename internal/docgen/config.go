package docgen

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ConfigField is one kongtrol.yaml key, read from the schema by reflection.
type ConfigField struct {
	Path     string   // yaml path: "security.kill_switch.mode", "vpns.<name>.type", "policies[].via"
	Section  string   // top-level key
	Type     string   // string, int, bool, list of string, map of string to string
	Default  string   // from config.Defaults(), "" when none
	Required bool     // validate:"required"
	Values   []string // validate:"oneof=…"
}

// ConfigFields walks config.Config and returns every leaf field in schema
// order, plus one entry per top-level section (Type "section").
func ConfigFields() []ConfigField {
	defaults := reflect.ValueOf(config.Defaults())
	var out []ConfigField
	t := reflect.TypeOf(config.Config{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := yamlName(f)
		if name == "" {
			continue
		}
		out = append(out, ConfigField{Path: name, Section: name, Type: "section", Required: hasRule(f, "required")})
		walkField(name, name, f.Type, defaults.Field(i), &out)
	}
	return out
}

func walkField(section, path string, t reflect.Type, def reflect.Value, out *[]ConfigField) {
	switch {
	case t.Kind() == reflect.Map && t.Elem().Kind() == reflect.Struct:
		walkStruct(section, path+".<name>", t.Elem(), reflect.Value{}, out)
	case t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Struct:
		walkStruct(section, path+"[]", t.Elem(), reflect.Value{}, out)
	case t.Kind() == reflect.Struct:
		walkStruct(section, path, t, def, out)
	}
}

func walkStruct(section, path string, t reflect.Type, def reflect.Value, out *[]ConfigField) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := yamlName(f)
		if name == "" {
			continue
		}
		p := path + "." + name
		var d reflect.Value
		if def.IsValid() {
			d = def.Field(i)
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			walkField(section, p, ft, d, out)
			continue
		}
		if (ft.Kind() == reflect.Slice || ft.Kind() == reflect.Map) && ft.Elem().Kind() == reflect.Struct {
			walkField(section, p, ft, d, out)
			continue
		}
		field := ConfigField{Path: p, Section: section, Type: typeName(ft), Required: hasRule(f, "required")}
		if d.IsValid() && !d.IsZero() && d.Kind() != reflect.Pointer {
			field.Default = fmt.Sprint(d.Interface())
		}
		for _, rule := range strings.Split(f.Tag.Get("validate"), ",") {
			if v, ok := strings.CutPrefix(rule, "oneof="); ok {
				field.Values = strings.Fields(v)
			}
		}
		*out = append(*out, field)
	}
}

func yamlName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
	if name == "-" {
		return ""
	}
	return name
}

func hasRule(f reflect.StructField, rule string) bool {
	for _, r := range strings.Split(f.Tag.Get("validate"), ",") {
		if r == rule {
			return true
		}
	}
	return false
}

func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Slice:
		return "list of " + typeName(t.Elem())
	case reflect.Map:
		return "map of " + typeName(t.Key()) + " to " + typeName(t.Elem())
	default:
		return t.Kind().String()
	}
}

// ConfigMarkdown renders docs/CONFIG.md (es) or docs/CONFIG.en.md (en).
// Descriptions come from the per-field catalog in text.go; a test fails
// when a schema field has none (or the catalog names a field that no
// longer exists).
func ConfigMarkdown(lang i18n.Lang, s Stamp) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!-- %s -->\n\n", text(lang, "generated.config"))
	b.WriteString(logoLine)
	fmt.Fprintf(&b, "# %s\n\n", text(lang, "config.title"))
	fmt.Fprintf(&b, "%s %s · %s\n\n", text(lang, "version"), s.Version, s.Date)
	fmt.Fprintf(&b, "%s\n\n", text(lang, "config.intro"))
	head := fmt.Sprintf("| %s | %s | %s | %s |\n|---|---|---|---|\n",
		text(lang, "config.col.field"), text(lang, "config.col.type"), text(lang, "config.col.default"), text(lang, "config.col.description"))
	open := false
	for _, f := range ConfigFields() {
		desc, ok := fieldDesc(lang, f.Path)
		if !ok {
			desc = text(lang, "config.nodesc")
		}
		if f.Type == "section" {
			if open {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "## `%s`\n\n%s\n\n%s", f.Path, desc, head)
			open = true
			continue
		}
		if f.Required {
			desc = "**" + text(lang, "config.required") + ".** " + desc
		}
		if len(f.Values) > 0 {
			desc += " " + text(lang, "config.values") + ": `" + strings.Join(f.Values, "`, `") + "`."
		}
		def := "—"
		if f.Default != "" {
			def = "`" + f.Default + "`"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", f.Path, typeText(lang, f.Type), def, cell(desc))
	}
	fmt.Fprintf(&b, "\n%s\n", text(lang, "config.more"))
	return b.Bytes(), nil
}

func typeText(lang i18n.Lang, t string) string {
	if lang != i18n.ES {
		return t
	}
	r := strings.NewReplacer("list of ", "lista de ", "map of ", "mapa de ", " to ", " a ", "string", "texto", "bool", "booleano", "int", "entero")
	return r.Replace(t)
}

// MissingDescriptions lists schema fields without a description in some
// language, and descriptions of fields that no longer exist.
func MissingDescriptions() []string {
	var out []string
	fields := map[string]bool{}
	for _, f := range ConfigFields() {
		fields[f.Path] = true
		for _, lang := range i18n.Langs() {
			if _, ok := fieldDesc(lang, f.Path); !ok {
				out = append(out, fmt.Sprintf("config field %q has no %s description in internal/docgen/text.go", f.Path, lang))
			}
		}
	}
	for _, lang := range i18n.Langs() {
		for path := range fieldText[lang] {
			if !fields[path] {
				out = append(out, fmt.Sprintf("internal/docgen/text.go describes %q (%s), which is no longer in the schema", path, lang))
			}
		}
	}
	sort.Strings(out)
	return out
}
