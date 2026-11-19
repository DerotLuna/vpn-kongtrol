//go:build linux

package monitor

import (
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// tcpConnectionsSupported: /proc/net/tcp is readable without admin.
const tcpConnectionsSupported = true

// activeRemoteAddrs returns the remote addresses of TCP connections that
// are established or being opened (/proc/net/tcp and tcp6).
func activeRemoteAddrs() ([]netip.Addr, error) {
	conns, err := tcpConnections()
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(conns))
	for _, c := range conns {
		out = append(out, c.Remote.Addr())
	}
	return out, nil
}

// tcpConnections lists established / opening TCP connections.
func tcpConnections() ([]TCPConn, error) {
	var out []TCPConn
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		out = append(out, parseProcNetTCPConns(string(b))...)
	}
	return out, nil
}

// processNames maps the socket inodes of conns to process names, the way
// `ss -p` does: /proc/<pid>/fd links read "socket:[<inode>]". Only this
// user's processes can be read without admin; others stay unnamed.
func processNames(conns []TCPConn) map[TCPConn]string {
	want := map[uint64]bool{}
	for _, c := range conns {
		if c.Inode != 0 {
			want[c.Inode] = true
		}
	}
	names := map[TCPConn]string{}
	if len(want) == 0 {
		return names
	}
	byInode := map[uint64]string{}
	procs, _ := os.ReadDir("/proc")
	for _, p := range procs {
		if _, err := strconv.Atoi(p.Name()); err != nil {
			continue
		}
		dir := filepath.Join("/proc", p.Name())
		fds, err := os.ReadDir(filepath.Join(dir, "fd"))
		if err != nil {
			continue // another user's process
		}
		comm := ""
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(dir, "fd", fd.Name()))
			if err != nil || !strings.HasPrefix(link, "socket:[") {
				continue
			}
			ino, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]"), 10, 64)
			if err != nil || !want[ino] {
				continue
			}
			if comm == "" {
				b, _ := os.ReadFile(filepath.Join(dir, "comm"))
				comm = strings.TrimSpace(string(b))
			}
			byInode[ino] = comm
		}
	}
	for _, c := range conns {
		if n := byInode[c.Inode]; n != "" {
			names[c] = n
		}
	}
	return names
}
