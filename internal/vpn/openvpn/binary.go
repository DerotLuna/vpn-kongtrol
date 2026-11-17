package openvpn

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// ErrBinaryNotFound is returned by FindBinary when no openvpn executable
// could be located. Callers (preflight, wizard) translate it for the user.
var ErrBinaryNotFound = errors.New("openvpn binary not found")

// FindBinary resolves the openvpn executable to run.
//
// Resolution order:
//  1. override (the profile's binary_path), when non-empty — must exist.
//  2. "openvpn" on PATH.
//  3. platform-specific well-known locations (platformCandidates), e.g. the
//     Windows installer's registry key or Homebrew's sbin on macOS, which are
//     usually NOT on PATH.
func FindBinary(override string) (string, error) {
	if override != "" {
		if isRegularFile(override) {
			return override, nil
		}
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, override)
	}
	if p, err := exec.LookPath("openvpn"); err == nil {
		return p, nil
	}
	for _, p := range platformCandidates() {
		if p != "" && isRegularFile(p) {
			return p, nil
		}
	}
	return "", ErrBinaryNotFound
}

func isRegularFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
