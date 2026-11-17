//go:build darwin

package netstats

import (
	"context"
	"os/exec"
	"time"
)

// counters parses `netstat -ibn -I <if>`: getifaddrs needs cgo and the CLI
// is built with CGO_ENABLED=0. Needs no admin.
func counters(name string) (sent, received uint64, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/sbin/netstat", "-ibn", "-I", name).Output()
	if err != nil {
		return 0, 0, false
	}
	return parseNetstatIB(string(out), name)
}
