package doctor

import (
	"os"
	"path/filepath"
	"strings"
)

// Probes behind checkPathCopies; tests replace them.
var (
	pathEnv = func() string { return os.Getenv("PATH") }
	selfExe = func() (string, error) {
		exe, err := os.Executable()
		if err != nil {
			return "", err
		}
		return filepath.EvalSymlinks(exe)
	}
)

// pathCopies lists every file named name in the PATH directories, in PATH
// order (the first one is what a terminal runs), without repeating a file
// reachable through two entries.
func pathCopies(pathList, name string) []string {
	var out []string
	var seen []os.FileInfo
	for _, dir := range filepath.SplitList(pathList) {
		dir = strings.Trim(dir, `"`)
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, name)
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		dup := false
		for _, s := range seen {
			if os.SameFile(s, fi) {
				dup = true
				break
			}
		}
		if !dup {
			seen = append(seen, fi)
			out = append(out, p)
		}
	}
	return out
}

// sameFile reports whether a and b are the same file on disk.
func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}

// checkPathCopies warns when `kongtrol` in a terminal is not this copy, or
// when several copies sit in PATH: the first one wins, so an old manual
// copy (e.g. in a tools folder) shadows the installed Kongtrol and
// `kongtrol --version` shows an old version.
func (d *run) checkPathCopies() {
	label := d.t("cli.doctor.label.path")
	self, err := selfExe()
	if err != nil {
		return // nothing to compare with; not worth a line
	}
	copies := pathCopies(pathEnv(), "kongtrol"+filepath.Ext(self))
	hint := d.f("doctor.hint.path", filepath.Dir(self))
	switch {
	case len(copies) == 0:
		d.ok(label, d.t("doctor.path.missing"))
	case !sameFile(copies[0], self):
		d.warn(label, d.f("doctor.path.other", copies[0]), hint)
	case len(copies) > 1:
		d.warn(label, d.f("doctor.path.dupes", len(copies), strings.Join(copies, ", ")), hint)
	default:
		d.ok(label, d.f("doctor.path.ok", copies[0]))
	}
}
