package openvpn

import (
	"path/filepath"
	"strings"
)

// RunningWithConfig reports whether some openvpn process (OpenVPN GUI, a
// manual run, or a previous Kongtrol instance) is already using cfgPath.
// Starting a second instance on the same profile makes both fight over the
// same routes, so callers should refuse and tell the user.
// Best effort: returns false when the process list cannot be read.
func RunningWithConfig(cfgPath string) bool {
	if cfgPath == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(cfgPath))
	for _, line := range strings.Split(openvpnCommandLines(), "\n") {
		l := strings.ToLower(line)
		if strings.Contains(l, "openvpn") && commandLineUsesConfig(l, base) {
			return true
		}
	}
	return false
}

// commandLineUsesConfig matches the config file name as a whole path
// element, so "rluna.conf" does not match "rluna-aws.conf".
func commandLineUsesConfig(cmdline, base string) bool {
	for idx := 0; ; {
		i := strings.Index(cmdline[idx:], base)
		if i < 0 {
			return false
		}
		i += idx
		before := byte(' ')
		if i > 0 {
			before = cmdline[i-1]
		}
		after := byte(' ')
		if end := i + len(base); end < len(cmdline) {
			after = cmdline[end]
		}
		if isPathBoundary(before) && isPathBoundary(after) {
			return true
		}
		idx = i + 1
	}
}

func isPathBoundary(c byte) bool {
	switch c {
	case ' ', '"', '\'', '\\', '/', '\t', '\r', '\n':
		return true
	}
	return false
}
