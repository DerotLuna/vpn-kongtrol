package profileimport

import (
	"os"
	"path/filepath"
	"strings"
)

// PlainDir: dir is a real folder (no link, no junction).
func PlainDir(dir string) bool {
	fi, err := os.Lstat(dir)
	return err == nil && fi.IsDir() && plain(dir, fi)
}

// CleanLocalPath accepts an absolute path to a file on this machine and
// returns it cleaned. Refused: relative paths, any ".." element, NUL, and
// device or network namespaces (\\.\, \\?\, UNC shares, /dev, /proc,
// /sys), which are never a profile or key file, and a UNC path would make
// Windows contact another machine (and send it the user's credentials).
// Shared by a "path" import (the file the user picked), the folder picked
// for an .ovpn's files and the references an .ovpn names outside its
// folder (ADR-0112).
func CleanLocalPath(raw string) (string, bool) {
	p := strings.TrimSpace(raw)
	if p == "" || strings.ContainsRune(p, 0) || !filepath.IsAbs(p) {
		return "", false
	}
	for _, el := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if el == ".." {
			return "", false
		}
	}
	// NT object paths (\??\UNC\…, \??\GLOBALROOT\…) reach SMB or devices
	// too, and Windows treats them as absolute.
	for _, prefix := range []string{`\\`, "//", `/\`, `\/`, `\??\`, `/??/`} {
		if strings.HasPrefix(p, prefix) {
			return "", false
		}
	}
	if !plainLocalVolume(p) {
		return "", false
	}
	p = filepath.Clean(p)
	for _, dir := range []string{"/dev", "/proc", "/sys"} {
		if p == dir || strings.HasPrefix(p, dir+"/") {
			return "", false
		}
	}
	return p, true
}
