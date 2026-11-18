package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// fakeInstall creates dir/kongtrol.exe for each dir and returns their paths.
func fakeInstall(t *testing.T, dirs ...string) []string {
	t.Helper()
	var out []string
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(d, "kongtrol.exe")
		if err := os.WriteFile(p, []byte(d), 0o755); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestCheckPathCopies(t *testing.T) {
	root := t.TempDir()
	installed := filepath.Join(root, "Programs", "Kongtrol")
	old := filepath.Join(root, "tools")
	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	exes := fakeInstall(t, installed, old)
	self := exes[0]
	join := func(dirs ...string) string { return strings.Join(dirs, string(os.PathListSeparator)) }

	cases := []struct {
		name       string
		path       string
		wantStatus string
		wantDetail string
		wantHint   bool
	}{
		{"not in PATH", join(empty), StatusOK, "no está en el PATH", false},
		{"only the installed copy", join(empty, installed), StatusOK, "usa esta copia: " + self, false},
		{"same dir twice is one copy", join(installed, installed), StatusOK, "usa esta copia", false},
		{"old copy first wins", join(old, installed), StatusWarn, "usa otra copia: " + exes[1], true},
		{"installed first, old copy later", join(installed, old), StatusWarn, "Hay 2 copias", true},
	}
	prevP, prevS := pathEnv, selfExe
	t.Cleanup(func() { pathEnv, selfExe = prevP, prevS })
	selfExe = func() (string, error) { return self, nil }
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pathEnv = func() string { return c.path }
			d := &run{opt: Options{Lang: i18n.ES}, rep: Report{Checks: []Check{}}}
			d.checkPathCopies()
			if len(d.rep.Checks) != 1 {
				t.Fatalf("checks = %+v", d.rep.Checks)
			}
			got := d.rep.Checks[0]
			if got.Status != c.wantStatus || !strings.Contains(got.Detail, c.wantDetail) {
				t.Fatalf("got %s %q, want %s ~%q", got.Status, got.Detail, c.wantStatus, c.wantDetail)
			}
			if c.wantHint && !strings.Contains(got.Hint, installed) {
				t.Fatalf("hint %q must name the installed folder %s", got.Hint, installed)
			}
		})
	}
}
