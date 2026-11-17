// Package impact reads docs/impact.yml (feature → code globs → surfaces)
// and tells which user-facing surfaces a change probably left out of date.
//
// It is advisory: CI posts the result as a non-blocking PR comment
// (AGENTS.md "Change impact check"). Same idea as GitHub CODEOWNERS
// (path globs → owners, https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners)
// and dorny/paths-filter, but mapping code to the docs that describe it.
package impact

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Map is docs/impact.yml.
type Map struct {
	Internal []string           `yaml:"internal"`
	Features map[string]Feature `yaml:"features"`
}

// Feature is one entry of the map.
type Feature struct {
	Code     []string `yaml:"code"`
	Surfaces []string `yaml:"surfaces"`
}

// Path is the map's location relative to the repository root.
const Path = "docs/impact.yml"

// Load reads root/docs/impact.yml.
func Load(root string) (*Map, error) {
	b, err := os.ReadFile(filepath.Join(root, Path))
	if err != nil {
		return nil, err
	}
	var m Map
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", Path, err)
	}
	return &m, nil
}

// Match reports whether slash path p matches glob g: "*" and "?" stay
// inside one path segment, "**" spans any number of them.
func Match(g, p string) bool {
	var re strings.Builder
	re.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch c := g[i]; {
		case c == '*' && i+1 < len(g) && g[i+1] == '*':
			re.WriteString(".*")
			i++
		case c == '*':
			re.WriteString("[^/]*")
		case c == '?':
			re.WriteString("[^/]")
		default:
			re.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	re.WriteString("$")
	ok, _ := regexp.MatchString(re.String(), p)
	return ok
}

// SurfaceFile is the file part of "path#anchor".
func SurfaceFile(s string) string { f, _, _ := strings.Cut(s, "#"); return f }

// Hit is one feature touched by a change.
type Hit struct {
	Feature   string
	Code      []string // changed files that belong to the feature
	Updated   []string // surfaces whose file changed too
	Untouched []string // surfaces whose file did not change
}

// Analyze returns the features whose code is among changed (slash paths),
// sorted by name.
func (m *Map) Analyze(changed []string) []Hit {
	set := map[string]bool{}
	for _, c := range changed {
		set[c] = true
	}
	var hits []Hit
	for name, f := range m.Features {
		h := Hit{Feature: name}
		for _, c := range changed {
			for _, g := range f.Code {
				if Match(g, c) {
					h.Code = append(h.Code, c)
					break
				}
			}
		}
		if len(h.Code) == 0 {
			continue
		}
		for _, s := range f.Surfaces {
			if set[SurfaceFile(s)] {
				h.Updated = append(h.Updated, s)
			} else {
				h.Untouched = append(h.Untouched, s)
			}
		}
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Feature < hits[j].Feature })
	return hits
}

var anchorRe = regexp.MustCompile(`feature:([a-z0-9_]+)`)

// Excerpt returns the paragraph a surface points at: from its anchor line
// to the next anchor or heading (at most maxLines lines), or the first
// maxLines lines of a whole-file surface. ok is false when the file or the
// anchor is missing.
func Excerpt(root, surface string, maxLines int) (text string, ok bool) {
	file, anchor, hasAnchor := strings.Cut(surface, "#")
	fh, err := os.Open(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return "", false
	}
	defer fh.Close()
	var out []string
	in := !hasAnchor
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		isAnchor := anchorRe.MatchString(line) && (strings.Contains(line, "<!--") || strings.Contains(line, "//"))
		if !in {
			if isAnchor && hasName(line, anchor) {
				in = true
				out = append(out, fmt.Sprintf("%s:%d", file, n))
			}
			continue
		}
		if hasAnchor && len(out) > 1 && (isAnchor || strings.HasPrefix(line, "#")) {
			break
		}
		out = append(out, line)
		if len(out) > maxLines {
			break
		}
	}
	if !in {
		return "", false
	}
	return strings.Join(out, "\n"), true
}

func hasName(line, name string) bool {
	for _, m := range anchorRe.FindAllStringSubmatch(line, -1) {
		if m[1] == name {
			return true
		}
	}
	return false
}

// Problems lists surfaces whose file or anchor does not exist.
func (m *Map) Problems(root string) []string {
	var out []string
	for name, f := range m.Features {
		if len(f.Code) == 0 || len(f.Surfaces) == 0 {
			out = append(out, fmt.Sprintf("feature %q needs code and surfaces", name))
		}
		for _, s := range f.Surfaces {
			if _, ok := Excerpt(root, s, 1); !ok {
				out = append(out, fmt.Sprintf("feature %q: surface %s not found (missing file or feature: anchor)", name, s))
			}
		}
	}
	sort.Strings(out)
	return out
}

// Markdown renders the PR comment section for hits.
func Markdown(hits []Hit, problems []string) string {
	var b strings.Builder
	b.WriteString("### Impacto en documentación / Docs impact\n\n")
	pending := 0
	for _, h := range hits {
		pending += len(h.Untouched)
	}
	if len(hits) == 0 {
		b.WriteString("Ningún feature de `docs/impact.yml` cambió. / No mapped feature changed.\n")
	} else if pending == 0 {
		b.WriteString("Todas las superficies de los features tocados cambiaron también. / Every surface of the touched features changed too.\n")
	} else {
		b.WriteString("Cambiaste código de estos features pero no estas superficies. **Si no aplica, explica el motivo en el PR** (`no aplica: <motivo>`). / You changed these features' code but not these surfaces; if it doesn't apply, say why in the PR.\n\n")
		for _, h := range hits {
			if len(h.Untouched) == 0 {
				continue
			}
			fmt.Fprintf(&b, "- **%s** (%s)\n", h.Feature, strings.Join(limit(h.Code, 5), ", "))
			for _, s := range h.Untouched {
				fmt.Fprintf(&b, "  - [ ] `%s`\n", s)
			}
		}
	}
	if len(problems) > 0 {
		b.WriteString("\n<details><summary>docs/impact.yml</summary>\n\n")
		for _, p := range problems {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("\n</details>\n")
	}
	return b.String()
}

func limit(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return append(s[:n:n], fmt.Sprintf("+%d", len(s)-n))
}
