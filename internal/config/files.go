package config

import (
	"errors"
	"io/fs"
	"os"
)

// ErrKeyFileRootOnly is the ValidationError key for a profile file only
// the administrator can read (e.g. /etc/wireguard/wg0.conf, root:root 0600
// inside a 0700 folder). Kongtrol runs as the user, so it can't use it in
// place; the "what to do" line offers the one-time copy with permission
// (ADR-0058).
const ErrKeyFileRootOnly = "config.err.file_root_only"

// IsPermission reports whether err means "not allowed to read".
func IsPermission(err error) bool { return errors.Is(err, fs.ErrPermission) }

// CheckFileReadable returns a *ValidationError (ErrKeyFileRootOnly) when
// path exists but this user may not read it. A missing file, an empty path
// or any other error returns nil: other checks report those.
func CheckFileReadable(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err == nil {
		_ = f.Close()
		return nil
	}
	if IsPermission(err) {
		return &ValidationError{Key: ErrKeyFileRootOnly, Args: []any{path}}
	}
	return nil
}
