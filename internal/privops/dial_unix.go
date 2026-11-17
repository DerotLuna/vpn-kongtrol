//go:build linux || darwin

package privops

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// SocketPath is the helper's Unix socket (/run on Linux, /var/run on macOS).
var SocketPath = filepath.Join(runDir, "kongtrol-helper.sock")

// dialHelper verifies the socket is a root-owned socket in a root-owned
// directory before connecting, so no user can impersonate the helper.
func dialHelper(ctx context.Context) (net.Conn, error) {
	fi, err := os.Lstat(SocketPath)
	if err != nil {
		return nil, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if fi.Mode()&os.ModeSocket == 0 || !ok || st.Uid != 0 || !TrustedDir(filepath.Dir(SocketPath)) {
		return nil, fmt.Errorf("%w: %s is not a root-owned socket", ErrDenied, SocketPath)
	}
	var d net.Dialer
	return d.DialContext(ctx, "unix", SocketPath)
}
