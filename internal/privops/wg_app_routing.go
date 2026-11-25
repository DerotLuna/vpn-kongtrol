package privops

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"
)

const (
	AppRouteFallback = "prefer_destinations"
	AppRouteOverride = "all"
)

const maxWGAppExecutables = 128

// WGAppRoutingSupported reports availability of the native back end.
func WGAppRoutingSupported() bool { return supportsWGAppRouting() }

// WGAppRouting scopes a tunnel's default route to exact executable files.
type WGAppRouting struct {
	Mode        string   `json:"mode"`
	Executables []string `json:"executables"`
}

func prepareWGAppRouting(data []byte, req WGAppRouting) ([]byte, error) {
	if !supportsWGAppRouting() {
		return nil, ErrUnsupported
	}
	if req.Mode != AppRouteFallback && req.Mode != AppRouteOverride {
		return nil, invalid("invalid executable routing mode")
	}
	if len(req.Executables) == 0 || len(req.Executables) > maxWGAppExecutables {
		return nil, invalid("invalid number of executable paths")
	}
	for _, path := range req.Executables {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, "\x00\r\n") {
			return nil, invalid("executable routing requires absolute executable paths")
		}
	}
	if result := inspectWireGuardDefaultEgress(data); !result.Compatible {
		return nil, invalid("WireGuard executable routing: %s", result.Reason)
	}

	// Keep peer defaults for crypto-routing, but no host default or
	// host-wide DNS change should be installed by wg-quick.
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		value, _, _ := strings.Cut(line, "#")
		key, _, _ := strings.Cut(value, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		switch key {
		case "table", "dns":
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
		if key == "[interface]" {
			out.WriteString("Table = off\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type wgAppRoutingSession struct {
	close func() error
	keys  map[[3]uint64]struct{} // inode, device and authorized UID
}

func (l *Local) clearWGAppRouting(name string) error {
	s, ok := l.wgAppRouting[name]
	if !ok {
		return nil
	}
	if err := s.close(); err != nil {
		return err
	}
	delete(l.wgAppRouting, name)
	l.leases.release(resource{resWGAppRoute, name})
	return nil
}
