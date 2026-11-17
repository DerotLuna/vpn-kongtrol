// Package docgen generates the reference docs that must never drift from
// the code: the CLI reference (docs/CLI.md, docs/CLI.en.md, docs/cli.json)
// from the cobra command tree, and the configuration reference
// (docs/CONFIG.md, docs/CONFIG.en.md) from internal/config's schema.
//
// Generated files are written by `go generate ./cmd/kongtrol` (or
// `make docs`) and are never edited by hand. Approach, as used by mature
// projects:
//   - cobra's doc generator (github.com/spf13/cobra/doc), used by kubectl,
//     gh and hugo, walks the command tree to emit reference pages; we do the
//     same walk but render one translated page per language plus JSON for
//     the landing (cobra/doc only emits English headings, one file per
//     command).
//   - golden files checked by tests, "go test" failing with a "regenerate"
//     hint (e.g. the Go project's own cmd/dist/testdata, gofmt testdata,
//     kubernetes hack/verify-generated-docs.sh).
package docgen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Stamp is the version and date every document carries. A generated file
// keeps its stamp while its content is unchanged; any change bumps the
// minor version and sets today's date, so regenerating is deterministic.
type Stamp struct {
	Version string
	Date    string
}

// File is one generated file: its path relative to the repository root and
// a renderer that embeds the given stamp.
type File struct {
	Path   string
	Render func(Stamp) ([]byte, error)
}

// RegenerateHint is printed whenever a generated file is stale.
const RegenerateHint = "run `go generate ./...` (or `make docs`) and commit the result; never edit generated files by hand"

var (
	mdStamp   = regexp.MustCompile(`(?m)^(?:Versión|Version) (\d+\.\d+) · (\d{4}-\d{2}-\d{2})`)
	jsonStamp = regexp.MustCompile(`"version": "(\d+\.\d+)",\s*"date": "(\d{4}-\d{2}-\d{2})"`)
)

func parseStamp(b []byte) (Stamp, bool) {
	for _, re := range []*regexp.Regexp{mdStamp, jsonStamp} {
		if m := re.FindSubmatch(b); m != nil {
			return Stamp{Version: string(m[1]), Date: string(m[2])}, true
		}
	}
	return Stamp{}, false
}

func bump(s Stamp, today string) Stamp {
	major, minor, ok := strings.Cut(s.Version, ".")
	n, err := strconv.Atoi(minor)
	if !ok || err != nil {
		return Stamp{Version: "1.0", Date: today}
	}
	return Stamp{Version: major + "." + strconv.Itoa(n+1), Date: today}
}

// normalize ignores CRLF line endings a Windows checkout may add.
func normalize(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }

// Update writes f under root when its content changed (or it is missing),
// bumping its stamp. It reports whether the file was written.
func Update(root string, f File, today string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(f.Path))
	stamp := Stamp{Version: "1.0", Date: today}
	old, err := os.ReadFile(path)
	if err == nil {
		if s, ok := parseStamp(old); ok {
			same, err := f.Render(s)
			if err != nil {
				return false, err
			}
			if bytes.Equal(same, normalize(old)) {
				return false, nil
			}
			stamp = bump(s, today)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	out, err := f.Render(stamp)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0o644)
}

// Check reports an error when f under root is missing or differs from a
// fresh generation (with the file's own stamp).
func Check(root string, f File) error {
	old, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
	if err != nil {
		return fmt.Errorf("%s: %v — %s", f.Path, err, RegenerateHint)
	}
	s, ok := parseStamp(old)
	if !ok {
		return fmt.Errorf("%s: no version/date stamp — %s", f.Path, RegenerateHint)
	}
	fresh, err := f.Render(s)
	if err != nil {
		return err
	}
	if !bytes.Equal(fresh, normalize(old)) {
		return fmt.Errorf("%s is out of date (line %d differs) — %s", f.Path, firstDiffLine(fresh, normalize(old)), RegenerateHint)
	}
	return nil
}

func firstDiffLine(a, b []byte) int {
	la, lb := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return i + 1
		}
	}
	return min(len(la), len(lb)) + 1
}

// logoLine puts the logo top left, like the hand-written docs.
const logoLine = "<img src=\"../assets/brand/logo-kong-64.png\" alt=\"VPN Kongtrol\" width=\"48\" align=\"left\" />\n\n"
