package daemonctl

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/sockowner"
)

// Every connection to the daemon goes through DialContext, which checks who
// owns the other end before the first byte (the API token) is sent
// (ADR-0062): another user of this machine could bind the daemon's port
// while Kongtrol is stopped and wait for the token.

// ForeignListenerError: another user's program answers on the daemon's
// address. Nothing was sent to it.
type ForeignListenerError struct {
	Addr string // host:port
	// Lang, when set, makes Error() the translated message with its
	// "what to do" line (Spawner sets it).
	Lang i18n.Lang
}

func (e *ForeignListenerError) Error() string {
	if e.Lang != "" {
		return e.Localized(e.Lang)
	}
	return "another user's program is listening on " + e.Addr + "; the Kongtrol token was not sent"
}

// Localized is the message plus its "what to do" line in lang.
func (e *ForeignListenerError) Localized(lang i18n.Lang) string {
	return i18n.F(lang, "daemon.err.foreign_listener", e.Addr) + " · " +
		i18n.F(lang, "daemon.err.foreign_listener.fix", portOf(e.Addr))
}

func portOf(addr string) string {
	if _, p, err := net.SplitHostPort(addr); err == nil {
		return p
	}
	return addr
}

// Test seams.
var (
	checkOwner = sockowner.Check
	daemonPID  = readDaemonPID
)

// PIDPath returns ~/.kongtrol/run/kongtrol.pid, written by the daemon.
func PIDPath() string { return filepath.Join(StateDir(), "run", "kongtrol.pid") }

func readDaemonPID() (uint32, bool) {
	b, err := os.ReadFile(PIDPath())
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 32)
	return uint32(n), err == nil
}

// verifyListener decides whether c (already connected, nothing sent yet)
// may carry the token. A different user → *ForeignListenerError. A process
// this user may not inspect (Windows: another user's, or this user's own
// elevated daemon) passes only when it is the daemon of this user's PID
// file, which only this user can write. Unknown (macOS, table not
// readable) proceeds: the token rotates on every daemon start.
func verifyListener(c net.Conn) error {
	la, ok1 := c.LocalAddr().(*net.TCPAddr)
	ra, ok2 := c.RemoteAddr().(*net.TCPAddr)
	if !ok1 || !ok2 || !ra.IP.IsLoopback() {
		return nil
	}
	res, err := checkOwner(la.AddrPort(), ra.AddrPort())
	switch res.Verdict {
	case sockowner.Self:
		return nil
	case sockowner.Other:
		return &ForeignListenerError{Addr: ra.String()}
	case sockowner.Denied:
		if pid, ok := daemonPID(); ok && pid == res.PID {
			return nil
		}
		return &ForeignListenerError{Addr: ra.String()}
	}
	if err != nil && !errors.Is(err, sockowner.ErrUnsupported) {
		slog.Debug("daemonctl: listener owner not verified", "addr", ra.String(), "err", err)
	}
	return nil
}

var netDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

// DialContext dials the daemon and refuses a listener owned by another
// user (verifyListener). Use it for every connection that carries the
// token: HTTPClient does; WebSocket dialers set NetDialContext to it.
func DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c, err := netDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	if err := verifyListener(c); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

var transport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil // the daemon is on loopback: never through a proxy
	t.DialContext = DialContext
	return t
}()

// HTTPClient returns a client for the daemon API whose connections passed
// the listener owner check.
func HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: transport}
}

// CheckListener connects to base's host:port and reports a
// *ForeignListenerError when another user owns it (nil otherwise, also
// when nothing listens). Spawner uses it so `up` and the tray explain the
// conflict instead of failing to start a daemon on a taken port.
func CheckListener(base string) error {
	host := strings.TrimPrefix(strings.TrimPrefix(base, "http://"), "https://")
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := DialContext(ctx, "tcp", host)
	var fe *ForeignListenerError
	if errors.As(err, &fe) {
		return fe
	}
	if err == nil {
		_ = c.Close()
	}
	return nil
}
