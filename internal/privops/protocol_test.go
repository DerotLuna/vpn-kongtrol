package privops

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeOps records the last call and returns canned results.
type fakeOps struct {
	last string
	arg  any
	err  error
}

func (f *fakeOps) rec(op string, a any) error { f.last, f.arg = op, a; return f.err }
func (f *fakeOps) Ping(context.Context) (PingResult, error) {
	return PingResult{Version: "v-test", Protocol: ProtocolVersion}, f.rec(OpPing, nil)
}
func (f *fakeOps) AddRoute(_ context.Context, r Route) error    { return f.rec(OpRouteAdd, r) }
func (f *fakeOps) DeleteRoute(_ context.Context, r Route) error { return f.rec(OpRouteDelete, r) }
func (f *fakeOps) FlushRoutes(_ context.Context, i string) error {
	return f.rec(OpRouteFlush, i)
}
func (f *fakeOps) SetDNS(_ context.Context, r DNSRequest) error { return f.rec(OpDNSSet, r) }
func (f *fakeOps) RestoreDNS(_ context.Context, i string) error { return f.rec(OpDNSRestore, i) }
func (f *fakeOps) KillSwitchEnable(_ context.Context, r KillSwitchRequest) error {
	return f.rec(OpKillSwitchOn, r)
}
func (f *fakeOps) KillSwitchDisable(context.Context) error { return f.rec(OpKillSwitchOff, nil) }
func (f *fakeOps) WGInstallTunnel(_ context.Context, r WGInstallRequest) (WGTunnel, error) {
	return WGTunnel{Name: "wg0"}, f.rec(OpWGInstall, r)
}
func (f *fakeOps) WGRemoveTunnel(_ context.Context, r WGRemoveRequest) error {
	return f.rec(OpWGRemove, r)
}
func (f *fakeOps) WGSetAllowedIPs(_ context.Context, r WGAllowedIPsRequest) error {
	return f.rec(OpWGAllowedIPs, r)
}
func (f *fakeOps) WGStats(_ context.Context, r WGStatsRequest) (WGStats, error) {
	return WGStats{Peers: []WGPeerStats{{PublicKey: "k", LatestHandshake: 1700000000, RxBytes: 1, TxBytes: 2, PersistentKeepalive: 25}}}, f.rec(OpWGStats, r)
}
func (f *fakeOps) OpenVPNStart(_ context.Context, r OpenVPNStartRequest) (OpenVPNSession, error) {
	return OpenVPNSession{ID: "abc", PID: 42, MgmtAddr: "127.0.0.1:1", MgmtPassword: "pw"}, f.rec(OpOpenVPNStart, r)
}
func (f *fakeOps) OpenVPNStop(_ context.Context, id string) error { return f.rec(OpOpenVPNStop, id) }
func (f *fakeOps) OpenVPNStatus(_ context.Context, id string) (OpenVPNState, error) {
	return OpenVPNState{Running: true, LogTail: []string{"Initialization Sequence Completed"}}, f.rec(OpOpenVPNStatus, id)
}
func (f *fakeOps) WriteHostsBlock(_ context.Context, e []HostEntry) error {
	return f.rec(OpHostsWrite, e)
}
func (f *fakeOps) RenewLease(ctx context.Context) error { return f.rec(OpLeaseRenew, leaseFrom(ctx)) }
func (f *fakeOps) SetBlockedDestinations(_ context.Context, r BlockRequest) error {
	return f.rec(OpBlockSet, r)
}

// pipeClient returns a Client whose every dial is served by Dispatch(ops)
// on the other end of a net.Pipe.
func pipeClient(ops Ops) *Client {
	return NewClientWithDialer(func(ctx context.Context) (net.Conn, error) {
		c, s := net.Pipe()
		go func() {
			defer s.Close()
			var req Request
			if err := ReadFrame(s, &req); err != nil {
				return
			}
			_ = WriteFrame(s, Dispatch(ctx, ops, req))
		}()
		return c, nil
	})
}

func TestClientRoundTrip(t *testing.T) {
	f := &fakeOps{}
	c := pipeClient(f)
	ctx := context.Background()

	res, err := c.Ping(ctx)
	if err != nil || res.Version != "v-test" {
		t.Fatalf("ping: %v %v", res, err)
	}
	r := Route{Destination: netip.MustParsePrefix("10.0.0.0/8"), Gateway: netip.MustParseAddr("10.8.0.1"), Interface: "wg0", Metric: 3}
	if err := c.AddRoute(ctx, r); err != nil || f.last != OpRouteAdd || f.arg.(Route) != r {
		t.Fatalf("add route: %v %v %v", err, f.last, f.arg)
	}
	noGW := Route{Destination: netip.MustParsePrefix("10.0.0.0/8"), Interface: "wg0"}
	if err := c.DeleteRoute(ctx, noGW); err != nil || f.arg.(Route).Gateway.IsValid() {
		t.Fatalf("zero gateway must round-trip as zero: %v %v", err, f.arg)
	}
	sess, err := c.OpenVPNStart(ctx, OpenVPNStartRequest{ConfigPath: "/x.ovpn"})
	if err != nil || sess.MgmtPassword != "pw" || sess.PID != 42 {
		t.Fatalf("openvpn start: %v %v", sess, err)
	}
	st, err := c.OpenVPNStatus(ctx, "abc")
	if err != nil || !st.Running || f.arg.(string) != "abc" {
		t.Fatalf("openvpn status: %v %v", st, err)
	}
	if err := c.WriteHostsBlock(ctx, []HostEntry{{IP: netip.MustParseAddr("10.0.0.1"), Host: "a.corp"}}); err != nil || f.last != OpHostsWrite {
		t.Fatalf("hosts: %v", err)
	}
	for _, call := range []func() error{
		func() error { return c.FlushRoutes(ctx, "wg0") },
		func() error { return c.SetDNS(ctx, DNSRequest{Interface: "wg0"}) },
		func() error { return c.RestoreDNS(ctx, "wg0") },
		func() error { return c.KillSwitchEnable(ctx, KillSwitchRequest{AllowLAN: true}) },
		func() error { return c.KillSwitchDisable(ctx) },
		func() error { _, err := c.WGInstallTunnel(ctx, WGInstallRequest{ConfigPath: "/a.conf"}); return err },
		func() error { return c.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "wg0"}) },
		func() error { return c.WGSetAllowedIPs(ctx, WGAllowedIPsRequest{Interface: "wg0"}) },
		func() error { return c.OpenVPNStop(ctx, "abc") },
	} {
		if err := call(); err != nil {
			t.Fatalf("call after %s: %v", f.last, err)
		}
	}
}

func TestClientErrorMapping(t *testing.T) {
	for _, sentinel := range []error{ErrInvalidRequest, ErrDenied, ErrNotFound, ErrUnsupported, ErrFailed} {
		f := &fakeOps{err: errors.Join(sentinel, errors.New("detail"))}
		err := pipeClient(f).KillSwitchDisable(context.Background())
		if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "detail") {
			t.Errorf("%v: got %v", sentinel, err)
		}
	}
	// A transport failure means the helper is not reachable.
	c := NewClientWithDialer(func(context.Context) (net.Conn, error) { return nil, os.ErrNotExist })
	if _, err := c.Ping(context.Background()); !errors.Is(err, ErrHelperNotInstalled) {
		t.Errorf("dial failure: %v", err)
	}
	if key, ok := GuidanceKey(ErrHelperNotInstalled); !ok || key != "service.error.not_installed" {
		t.Errorf("guidance key: %q", key)
	}
}

func TestDispatchRejectsHostileRequests(t *testing.T) {
	f := &fakeOps{}
	ctx := context.Background()
	cases := map[string]Request{
		"old version":   {V: 0, Op: OpPing},
		"unknown op":    {V: ProtocolVersion, Op: "exec"},
		"unknown field": {V: ProtocolVersion, Op: OpRouteFlush, Args: []byte(`{"interface":"wg0","cmd":"x"}`)},
		"trailing":      {V: ProtocolVersion, Op: OpRouteFlush, Args: []byte(`{"interface":"wg0"}{}`)},
		"wrong type":    {V: ProtocolVersion, Op: OpRouteAdd, Args: []byte(`{"destination":5}`)},
		"bad prefix":    {V: ProtocolVersion, Op: OpRouteAdd, Args: []byte(`{"destination":"10.0.0.0/99"}`)},
		"args on ping":  {V: ProtocolVersion, Op: OpPing, Args: []byte(`{"x":1}`)},
	}
	for name, req := range cases {
		f.last = ""
		resp := Dispatch(ctx, f, req)
		if resp.OK || f.last != "" {
			t.Errorf("%s: accepted (resp=%+v last=%q)", name, resp, f.last)
		}
	}
}

func TestFrameLimits(t *testing.T) {
	var buf bytes.Buffer
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, MaxFrame+1)
	buf.Write(hdr)
	if err := ReadFrame(&buf, &Request{}); err == nil {
		t.Error("oversize frame accepted")
	}
	binary.BigEndian.PutUint32(hdr, 0)
	if err := ReadFrame(bytes.NewReader(hdr), &Request{}); err == nil {
		t.Error("empty frame accepted")
	}
	if err := WriteFrame(&buf, strings.Repeat("x", MaxFrame)); err == nil {
		t.Error("oversize write accepted")
	}
	buf.Reset()
	_ = WriteFrame(&buf, Request{V: 1, Op: OpPing})
	var got Request
	if err := ReadFrame(&buf, &got); err != nil || got.Op != OpPing {
		t.Errorf("round trip: %v %v", got, err)
	}
}

// Local must reject bad input before touching the system.
func TestLocalValidatesBeforeSystemCalls(t *testing.T) {
	interfaceExists = func(string) bool { return false }
	defer func() { interfaceExists = func(string) bool { return true } }()
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	ctx := context.Background()

	if err := l.AddRoute(ctx, Route{Destination: netip.MustParsePrefix("10.0.0.0/8"), Interface: "nope0"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing interface: %v", err)
	}
	if err := l.AddRoute(ctx, Route{Interface: "wg0"}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("invalid route: %v", err)
	}
	if err := l.SetDNS(ctx, DNSRequest{Interface: "nope0", Servers: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}); !errors.Is(err, ErrNotFound) {
		t.Errorf("dns missing iface: %v", err)
	}
	if err := l.KillSwitchEnable(ctx, KillSwitchRequest{Interfaces: []string{"nope0"}}); !errors.Is(err, ErrNotFound) {
		t.Errorf("ks missing iface: %v", err)
	}
	if err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "someones-tunnel"}); !errors.Is(err, ErrDenied) {
		t.Errorf("unmanaged wg remove: %v", err)
	}
	key := "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
	if err := l.WGSetAllowedIPs(ctx, WGAllowedIPsRequest{Interface: "other", PeerPublicKey: key,
		AllowedIPs: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}); !errors.Is(err, ErrDenied) {
		t.Errorf("unmanaged wg set: %v", err)
	}
	if _, err := l.OpenVPNStatus(ctx, randomHex(16)); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown session: %v", err)
	}
	if err := l.OpenVPNStop(ctx, "../../etc"); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("bad session id: %v", err)
	}
	if _, err := l.OpenVPNStart(ctx, OpenVPNStartRequest{ConfigPath: "relative.ovpn"}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("relative path: %v", err)
	}
	if err := l.WriteHostsBlock(ctx, []HostEntry{{IP: netip.MustParseAddr("10.0.0.1"), Host: "Bad Host"}}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("bad host entry: %v", err)
	}

	// Config files are read by the helper itself and rejected on content.
	dir := t.TempDir()
	ovpn := filepath.Join(dir, "evil.ovpn")
	if err := os.WriteFile(ovpn, []byte(baseOVPN+"up /tmp/x.sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.OpenVPNStart(ctx, OpenVPNStartRequest{ConfigPath: ovpn}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("dangerous ovpn: %v", err)
	}
	wg := filepath.Join(dir, "wg0.conf")
	if err := os.WriteFile(wg, []byte("[Interface]\nPrivateKey = k\nPostUp = id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.WGInstallTunnel(ctx, WGInstallRequest{ConfigPath: wg}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("dangerous wg conf: %v", err)
	}
	if _, err := l.WGInstallTunnel(ctx, WGInstallRequest{ConfigPath: filepath.Join(dir, "missing.conf")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing wg conf: %v", err)
	}
}

func TestReadTrustedFileOwnFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ok.ovpn")
	if err := os.WriteFile(p, []byte("client\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _, err := readTrustedFile(selfCaller(), p, 100, nil)
	if err != nil || string(b) != "client\n" {
		t.Fatalf("own file: %q %v", b, err)
	}
	if _, _, err := readTrustedFile(selfCaller(), p, 3, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("size cap: %v", err)
	}
	if _, _, err := readTrustedFile(selfCaller(), filepath.Dir(p), 100, nil); err == nil {
		t.Error("directory accepted")
	}
}
