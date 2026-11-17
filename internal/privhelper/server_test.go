package privhelper

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// stubOps embeds a Local (never reached for these ops) and overrides Ping.
type stubOps struct {
	privops.Ops
	calls int
}

func (s *stubOps) Ping(ctx context.Context) (privops.PingResult, error) {
	s.calls++
	return privops.PingResult{Version: "test", Protocol: privops.ProtocolVersion, Caller: privops.CallerFrom(ctx).Name}, nil
}

func newTestServer(t *testing.T, auth Authenticator) (*Server, *stubOps, string) {
	t.Helper()
	ops := &stubOps{Ops: privops.NewLocal(privops.LocalOptions{StateDir: t.TempDir()})}
	logPath := filepath.Join(t.TempDir(), "audit.log")
	return &Server{Ops: ops, Authenticate: auth, Audit: OpenAudit(logPath)}, ops, logPath
}

func clientFor(s *Server) *privops.Client {
	return privops.NewClientWithDialer(func(ctx context.Context) (net.Conn, error) {
		c, srv := net.Pipe()
		go s.HandleConn(ctx, srv)
		return c, nil
	})
}

func allowAs(name string) Authenticator {
	return func(net.Conn) (privops.Caller, func(), error) {
		return privops.Caller{ID: "1000", Name: name, UID: 1000}, nil, nil
	}
}

func TestServerRoundTripAndAudit(t *testing.T) {
	s, ops, logPath := newTestServer(t, allowAs("alice"))
	c := clientFor(s)
	res, err := c.Ping(context.Background())
	if err != nil || res.Caller != "alice" || ops.calls != 1 {
		t.Fatalf("ping: %+v %v", res, err)
	}
	// Validation happens server-side even if the client skipped it.
	err = c.AddRoute(context.Background(), privops.Route{Interface: "wg0;reboot"})
	if !errors.Is(err, privops.ErrInvalidRequest) {
		t.Fatalf("invalid route: %v", err)
	}
	log, _ := os.ReadFile(logPath)
	for _, want := range []string{`"op":"ping"`, `"caller":"alice"`, `"op":"route.add"`, `"code":"invalid"`} {
		if !strings.Contains(string(log), want) {
			t.Errorf("audit log missing %s:\n%s", want, log)
		}
	}
}

func TestServerDeniesUnauthenticated(t *testing.T) {
	s, ops, logPath := newTestServer(t, func(net.Conn) (privops.Caller, func(), error) {
		return privops.Caller{}, nil, errors.New("uid 1001 not allowed")
	})
	_, err := clientFor(s).Ping(context.Background())
	if !errors.Is(err, privops.ErrDenied) || ops.calls != 0 {
		t.Fatalf("want denied, got %v (calls=%d)", err, ops.calls)
	}
	if log, _ := os.ReadFile(logPath); !strings.Contains(string(log), "uid 1001 not allowed") {
		t.Errorf("denial not audited: %s", log)
	}
}

func TestServerRejectsMalformedFrames(t *testing.T) {
	s, ops, _ := newTestServer(t, allowAs("alice"))
	for name, payload := range map[string][]byte{
		"oversize": func() []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, privops.MaxFrame+1); return b }(),
		"not json": append([]byte{0, 0, 0, 3}, []byte("abc")...),
		"unknown":  append([]byte{0, 0, 0, 23}, []byte(`{"v":1,"op":"ping","x":1}`)[:23]...),
	} {
		c, srv := net.Pipe()
		go s.HandleConn(context.Background(), srv)
		go func() { _, _ = c.Write(payload) }()
		var resp privops.Response
		if err := privops.ReadFrame(c, &resp); err != nil || resp.OK {
			t.Errorf("%s: resp=%+v err=%v", name, resp, err)
		}
		c.Close()
	}
	if ops.calls != 0 {
		t.Errorf("malformed frames reached ops")
	}
}

func TestAuthorize(t *testing.T) {
	cfg := Config{UserID: "S-1-5-21-1-2-3-1001"}
	if authorize(cfg, privops.Caller{ID: "S-1-5-21-1-2-3-1001"}) != nil {
		t.Error("installing user denied")
	}
	if authorize(cfg, privops.Caller{ID: "S-1-5-21-9-9-9-1002", Admin: true}) != nil {
		t.Error("admin denied")
	}
	if !errors.Is(authorize(cfg, privops.Caller{ID: "S-1-5-21-9-9-9-1002"}), privops.ErrDenied) {
		t.Error("other user allowed")
	}
	if !errors.Is(authorize(Config{}, privops.Caller{}), privops.ErrDenied) {
		t.Error("empty identity allowed")
	}
}

func TestConfigValidation(t *testing.T) {
	if _, err := encodeConfig(Config{Format: ConfigFormat, UserID: "not a user"}); err == nil {
		t.Error("bad user id accepted")
	}
	if _, err := encodeConfig(Config{Format: 99, UserID: sampleUserID}); err == nil {
		t.Error("bad format accepted")
	}
	if _, err := encodeConfig(Config{Format: ConfigFormat, UserID: sampleUserID}); err != nil {
		t.Error(err)
	}
}
