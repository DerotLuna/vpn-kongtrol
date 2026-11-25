//go:build !windows && !darwin

package profileimport

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// SearchRoots are the folders searched for a dropped file's original
// (ADR-0112): Downloads, Desktop and Documents as the XDG user-dirs file
// names them (localized names included; ~/Downloads… when it is absent),
// ~/.config/openvpn, and /etc/openvpn/client and /etc/openvpn, which are
// only entered when the user can read them (never with root).
func SearchRoots() []string {
	h := home()
	if h == "" {
		return existing("/etc/openvpn/client", "/etc/openvpn")
	}
	dirs := xdgUserDirs(h)
	pick := func(key, fallback string) string {
		if d := dirs[key]; d != "" {
			return d
		}
		return filepath.Join(h, fallback)
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(cfg) {
		cfg = filepath.Join(h, ".config")
	}
	return existing(
		pick("XDG_DOWNLOAD_DIR", "Downloads"), pick("XDG_DESKTOP_DIR", "Desktop"), pick("XDG_DOCUMENTS_DIR", "Documents"),
		filepath.Join(cfg, "openvpn"), "/etc/openvpn/client", "/etc/openvpn",
	)
}

// xdgUserDirs reads $XDG_CONFIG_HOME/user-dirs.dirs (xdg-user-dirs): lines
// XDG_<NAME>_DIR="$HOME/<path>" or an absolute path; anything else is
// ignored (the format allows only those two forms).
func xdgUserDirs(h string) map[string]string {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(cfg) {
		cfg = filepath.Join(h, ".config")
	}
	out := map[string]string{}
	f, err := os.Open(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || strings.HasPrefix(k, "#") {
			continue
		}
		v = strings.Trim(v, `"`)
		switch {
		case strings.HasPrefix(v, "$HOME/"):
			v = filepath.Join(h, strings.TrimPrefix(v, "$HOME/"))
		case !filepath.IsAbs(v):
			continue
		}
		if filepath.Clean(v) == filepath.Clean(h) { // "disabled" dirs point at $HOME
			continue
		}
		out[k] = v
	}
	return out
}
