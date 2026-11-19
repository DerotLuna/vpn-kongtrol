package profileimport

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ErrKeyProtectedPath is the one refusal of the elevated read: the path is
// not absolute, not .conf/.ovpn, not a regular file with a single link
// (a symlink, a hard link, a folder, a device), swapped while it was
// opened, too big, or not a WireGuard/OpenVPN profile. It never says which
// check failed nor quotes the file (ADR-0063).
const ErrKeyProtectedPath = "api.err.import_protected_path"

// ReadProtected reads a profile file only the administrator can read
// (e.g. /etc/wireguard/wg0.conf). It is what `kongtrol profile
// read-protected` (hidden) runs as root after one OS permission prompt (sudo in a
// terminal, pkexec from the dashboard), ADR-0058. Nothing leaves it unless
// the content passes the helper's own checks (Validate) and has a profile's
// shape: a WireGuard [Interface] with a PrivateKey and only WireGuard keys,
// or a self-contained .ovpn without directives that run programs, with at
// least one `remote` and `client` or `tls-client`. So the prompt can only
// ever release a VPN profile, never another root-only file.
//
// Checks: absolute path; .conf or .ovpn; the last element is not a link
// (Lstat) and is a regular file; the opened file is that same file
// (os.SameFile, no swap between check and open) and has exactly one link
// (a hard link could give a root-only file another name; Unix only); at
// most MaxSize bytes.
//
// Every refusal is the same generic ErrKeyProtectedPath with the path the
// caller typed: the validators' detailed errors name sections, keys or a
// line of the file, and this output crosses the root boundary (ADR-0063).
// The normal import of a file the user can read keeps the detailed errors.
func ReadProtected(path string) ([]byte, error) {
	refuse := &config.ValidationError{Key: ErrKeyProtectedPath, Args: []any{path}}
	typ := TypeForFile(path)
	if !filepath.IsAbs(path) || typ == "" {
		return nil, refuse
	}
	path = filepath.Clean(path)
	before, err := os.Lstat(path)
	if err != nil {
		return nil, refuse
	}
	if !before.Mode().IsRegular() {
		return nil, refuse
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, refuse
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() {
		return nil, refuse
	}
	if n, known := linkCount(after); known && n != 1 {
		return nil, refuse
	}
	if after.Size() > MaxSize {
		return nil, refuse
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil || int64(len(data)) > MaxSize {
		return nil, refuse
	}
	if err := Validate(typ, data); err != nil {
		return nil, refuse
	}
	if typ == "openvpn" && !openVPNProfileShape(data) {
		return nil, refuse
	}
	return data, nil
}

// openVPNProfileShape reports a client profile: at least one `remote` (top
// level or inside <connection>) and `client` or `tls-client`. The OpenVPN
// validator is a denylist (unknown lines pass), so without this any text
// file named .ovpn would be released. Inline blocks (<ca>…) are skipped.
func openVPNProfileShape(data []byte) bool {
	remote, client := false, false
	inTag := ""
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), int(MaxSize)+1)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if strings.HasPrefix(line, "<") && strings.HasSuffix(line, ">") {
			tag := strings.ToLower(strings.Trim(line, "<>"))
			switch {
			case strings.HasPrefix(tag, "/"):
				inTag = ""
			case tag != "connection":
				inTag = tag
			}
			continue
		}
		if inTag != "" {
			continue // certificate or key text
		}
		switch strings.ToLower(strings.TrimPrefix(strings.Fields(line)[0], "--")) {
		case "remote":
			remote = true
		case "client", "tls-client":
			client = true
		}
	}
	return sc.Err() == nil && remote && client
}
