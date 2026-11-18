// Package configexport writes a loaded kongtrol.yaml back out as a
// shareable template ("kongtrol export", GET /api/v1/config/export, the
// dashboard's "Exportar configuración").
//
// The output is the config marshalled by the same YAML library the loader
// uses, so it always parses back to the same config: lists stay lists,
// strings such as "*.corp.com" are quoted, every field is kept and keys
// come out in a stable order (struct order, map keys sorted). The config
// holds no secrets — passwords live in the OS keychain — so nothing has to
// be redacted; a translated header says how to restore them.
package configexport

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// Style colours the output for a terminal. Nil functions leave the text
// as is; the zero Style writes plain, valid YAML.
type Style struct {
	Section func(string) string // top-level keys (vpns, policies, …)
	Key     func(string) string
	Value   func(string) string
	Comment func(string) string
}

// Filename is the name offered when the export is downloaded.
const Filename = "kongtrol-export.yaml"

var headerKeys = []string{
	"export.header.title",
	"export.header.secrets",
	"export.header.paths",
	"export.header.import",
}

// Write writes cfg as YAML with a translated comment header.
func Write(w io.Writer, cfg *config.Config, lang i18n.Lang, style Style) error {
	if cfg == nil {
		return fmt.Errorf("configexport: no config")
	}
	var body bytes.Buffer
	enc := yaml.NewEncoder(&body)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("configexport: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("configexport: %w", err)
	}

	bw := bufio.NewWriter(w)
	for _, k := range headerKeys {
		fmt.Fprintln(bw, apply(style.Comment, "# "+i18n.T(lang, k)))
	}
	fmt.Fprintln(bw)
	sc := bufio.NewScanner(&body)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		fmt.Fprintln(bw, colorLine(sc.Text(), style))
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return bw.Flush()
}

func apply(fn func(string) string, s string) string {
	if fn == nil || s == "" {
		return s
	}
	return fn(s)
}

// colorLine styles one line of block YAML as produced by yaml.v3: an
// optional "- " list marker, then "key:" with an optional value, or a
// bare list value. Quoted scalars are never split.
func colorLine(line string, st Style) string {
	if st.Section == nil && st.Key == nil && st.Value == nil && st.Comment == nil {
		return line
	}
	trimmed := strings.TrimLeft(line, " ")
	indent := line[:len(line)-len(trimmed)]
	dash := ""
	if strings.HasPrefix(trimmed, "- ") {
		dash, trimmed = "- ", trimmed[2:]
	}
	if strings.HasPrefix(trimmed, "#") {
		return indent + dash + apply(st.Comment, trimmed)
	}
	if !strings.HasPrefix(trimmed, `"`) && !strings.HasPrefix(trimmed, "'") {
		if i := keyEnd(trimmed); i > 0 {
			key, rest := trimmed[:i], trimmed[i+1:]
			keyFn := st.Key
			if indent == "" && dash == "" && st.Section != nil {
				keyFn = st.Section
			}
			out := indent + dash + apply(keyFn, key) + apply(st.Key, ":")
			if v := strings.TrimPrefix(rest, " "); v != "" {
				out += " " + apply(st.Value, v)
			}
			return out
		}
	}
	return indent + dash + apply(st.Value, trimmed)
}

// keyEnd is the index of the ':' that ends a plain mapping key, or -1.
func keyEnd(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' && (i+1 == len(s) || s[i+1] == ' ') {
			return i
		}
		if s[i] == ' ' {
			return -1
		}
	}
	return -1
}
