package vpndetect

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Provider apps (Proton VPN, NordVPN, Surfshark) are found from a list of
// candidate install folders: the installer's default folder plus the
// InstallLocation of the app's entry in Windows "Apps & features"
// (uninstall registry keys), so a custom install folder is found too.
// The pure parts live here so they are tested on every OS; the Windows
// folder lists are in apps_windows.go.

// AppLayout describes where an app keeps its executable inside one
// install folder.
type AppLayout struct {
	// Exes are file names directly in the install folder, in preference
	// order (e.g. Proton's ProtonVPN.Launcher.exe).
	Exes []string
	// VersionedExes are file names inside the newest "v4.1.10" / "7.24.5.0"
	// style subfolder (Proton keeps the client in "<dir>\v<version>\").
	VersionedExes []string
}

// AppPaths lists the executables of layout found in dirs, folder by
// folder, without duplicates. Missing folders are skipped. Nothing is run.
func AppPaths(dirs []string, layout AppLayout) []string {
	var out []string
	add := func(p string) {
		if !slices.ContainsFunc(out, func(q string) bool { return samePath(p, q) }) {
			out = append(out, p)
		}
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		for _, name := range layout.Exes {
			if p := filepath.Join(dir, name); isFile(p) {
				add(p)
			}
		}
		if len(layout.VersionedExes) == 0 {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var subdirs []string
		for _, e := range entries {
			if e.IsDir() {
				subdirs = append(subdirs, e.Name())
			}
		}
		// Newest version first; an older leftover folder is only a fallback.
		for _, sub := range VersionDirsNewestFirst(subdirs) {
			for _, name := range layout.VersionedExes {
				if p := filepath.Join(dir, sub, name); isFile(p) {
					add(p)
				}
			}
		}
	}
	return out
}

// VersionDirsNewestFirst keeps the names that are versions ("v4.1.10",
// "7.24.5.0") and orders them newest first, comparing numbers (v4.10 is
// newer than v4.9). Ties keep their input order.
func VersionDirsNewestFirst(names []string) []string {
	type vdir struct {
		name string
		v    []int
	}
	var vs []vdir
	for _, n := range names {
		if v, ok := parseVersion(n); ok {
			vs = append(vs, vdir{n, v})
		}
	}
	slices.SortStableFunc(vs, func(a, b vdir) int { return compareVersions(b.v, a.v) })
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.name
	}
	return out
}

// parseVersion reads "v4.1.10", "V4.1" or "7.24.5.0": an optional leading
// v, then 1 to 4 dot-separated non-negative integers.
func parseVersion(s string) ([]int, bool) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	parts := strings.Split(s, ".")
	if s == "" || len(parts) > 4 {
		return nil, false
	}
	v := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p == "" || p[0] == '+' {
			return nil, false
		}
		v[i] = n
	}
	return v, true
}

// compareVersions compares numerically; missing parts count as 0.
func compareVersions(a, b []int) int {
	for i := range max(len(a), len(b)) {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// comparePaths orders paths segment by segment: version-like segments
// numerically, the rest case-insensitively. Detect uses it to pick the
// newest of several matches (a plain string sort puts v4.9 after v4.10).
func comparePaths(a, b string) int {
	as := strings.FieldsFunc(a, isSep)
	bs := strings.FieldsFunc(b, isSep)
	for i := range min(len(as), len(bs)) {
		va, oka := parseVersion(as[i])
		vb, okb := parseVersion(bs[i])
		if oka && okb {
			if c := compareVersions(va, vb); c != 0 {
				return c
			}
			continue
		}
		if c := strings.Compare(strings.ToLower(as[i]), strings.ToLower(bs[i])); c != 0 {
			return c
		}
	}
	return len(as) - len(bs)
}

func isSep(r rune) bool { return r == '/' || r == '\\' }

// InstallDirFromEntry returns the install folder of an uninstall registry
// entry: InstallLocation, else the folder of DisplayIcon
// (`"C:\…\App.exe",0` is common). "" when neither gives an absolute path.
func InstallDirFromEntry(installLocation, displayIcon string) string {
	if d := strings.Trim(strings.TrimSpace(installLocation), `"`); d != "" && isAbsAnyOS(d) {
		return strings.TrimRight(d, `\/`)
	}
	icon := strings.TrimSpace(displayIcon)
	if i := strings.LastIndex(icon, ","); i > 0 && !strings.ContainsAny(icon[i:], `\/`) {
		icon = icon[:i]
	}
	icon = strings.Trim(strings.TrimSpace(icon), `"`)
	if icon == "" || !isAbsAnyOS(icon) {
		return ""
	}
	if i := strings.LastIndexAny(icon, `\/`); i > 0 {
		return icon[:i]
	}
	return ""
}

// isAbsAnyOS accepts Windows drive / UNC paths on every OS, so the
// registry parsing is tested everywhere.
func isAbsAnyOS(p string) bool {
	if len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') {
		return true
	}
	return strings.HasPrefix(p, `\\`) || filepath.IsAbs(p)
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
