package profileimport

import (
	"io"
	"os"
	"path/filepath"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ErrKeyProtectedPath refuses a path the elevated read never opens: not
// absolute, not .conf/.ovpn, not a regular file (a link, a folder, a
// device) or swapped while it was opened.
const ErrKeyProtectedPath = "api.err.import_protected_path"

// ReadProtected reads a profile file only the administrator can read
// (e.g. /etc/wireguard/wg0.conf). It is what `kongtrol profile
// read-protected` (hidden) runs as root after one OS permission prompt (sudo in a
// terminal, pkexec from the dashboard), ADR-0058. Nothing leaves it unless
// the content passes the helper's own checks (Validate): a WireGuard
// [Interface] with a PrivateKey and only WireGuard keys, or a
// self-contained .ovpn without directives that run programs. So the
// prompt can only ever release a VPN profile, never another root-only
// file (a refusal names the line or key, never echoes the content).
//
// Checks: absolute path; .conf or .ovpn; the last element is not a link
// (Lstat) and is a regular file; the opened file is that same file
// (os.SameFile, no swap between check and open); at most MaxSize bytes.
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
	if err != nil || !os.SameFile(before, after) {
		return nil, refuse
	}
	if after.Size() > MaxSize {
		return nil, TooBig()
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return nil, refuse
	}
	if int64(len(data)) > MaxSize {
		return nil, TooBig()
	}
	if err := Validate(typ, data); err != nil {
		return nil, err
	}
	return data, nil
}
