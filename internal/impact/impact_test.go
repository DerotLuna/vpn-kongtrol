package impact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const root = "../.."

func TestMatch(t *testing.T) {
	for _, c := range []struct {
		g, p string
		want bool
	}{
		{"internal/policy/**", "internal/policy/engine.go", true},
		{"internal/policy/**", "internal/policyx/a.go", false},
		{"internal/security/killswitch*.go", "internal/security/killswitch_priv.go", true},
		{"internal/vpn/*.go", "internal/vpn/openvpn/adapter.go", false},
		{"cmd/kongtrol/daemon_*.go", "cmd/kongtrol/daemon_cmd.go", true},
	} {
		if got := Match(c.g, c.p); got != c.want {
			t.Errorf("Match(%q, %q) = %v", c.g, c.p, got)
		}
	}
}

// TestEveryPackageMapped keeps docs/impact.yml from rotting: every
// top-level package under internal/ and cmd/ must belong to some feature
// or be listed under `internal` (no user-facing surface).
func TestEveryPackageMapped(t *testing.T) {
	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, top := range []string{"internal", "cmd"} {
		entries, err := os.ReadDir(filepath.Join(root, top))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			pkg := top + "/" + e.Name()
			if covered(m, pkg) {
				continue
			}
			t.Errorf("%s is not in docs/impact.yml: add it to a feature's code globs, or to `internal` if users never see it", pkg)
		}
	}
}

func covered(m *Map, pkg string) bool {
	for _, p := range m.Internal {
		if p == pkg {
			return true
		}
	}
	for _, f := range m.Features {
		for _, g := range f.Code {
			if g == pkg+"/**" || strings.HasPrefix(g, pkg+"/") {
				return true
			}
		}
	}
	return false
}

func TestAnalyze(t *testing.T) {
	m := &Map{Features: map[string]Feature{
		"ks": {Code: []string{"internal/security/killswitch*.go"}, Surfaces: []string{"README.md#ks", "docs/SECURITY.md"}},
	}}
	hits := m.Analyze([]string{"internal/security/killswitch.go", "docs/SECURITY.md"})
	if len(hits) != 1 || len(hits[0].Untouched) != 1 || hits[0].Untouched[0] != "README.md#ks" {
		t.Fatalf("hits = %+v", hits)
	}
}
