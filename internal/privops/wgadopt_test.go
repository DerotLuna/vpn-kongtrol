package privops

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

const adoptConf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.8.0.2/32

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
AllowedIPs = 10.0.0.0/8
Endpoint = vpn.example.com:51820
`

// wgStubs replaces the system hooks; isTunnel answers sysWGIsTunnel.
type wgStubs struct {
	downs, sets []string
}

func stubWG(t *testing.T, isTunnel func(string) (bool, error)) *wgStubs {
	t.Helper()
	s := &wgStubs{}
	oldIs, oldDown, oldSet, oldDir := wgIsTunnelFn, wgDownFn, wgSetFn, ensurePrivateDir
	t.Cleanup(func() { wgIsTunnelFn, wgDownFn, wgSetFn, ensurePrivateDir = oldIs, oldDown, oldSet, oldDir })
	ensurePrivateDir = func(d string) error { return os.MkdirAll(d, 0o700) }
	wgIsTunnelFn = func(_ context.Context, n string) (bool, error) { return isTunnel(n) }
	wgDownFn = func(_ context.Context, n, conf string) error {
		if _, err := os.Stat(conf); err != nil {
			t.Errorf("down %s without a private copy: %v", n, err)
		}
		s.downs = append(s.downs, n)
		return nil
	}
	wgSetFn = func(_ context.Context, r WGAllowedIPsRequest) error { s.sets = append(s.sets, r.Interface); return nil }
	return s
}

func profileConf(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name+".conf")
	if err := os.WriteFile(p, []byte(adoptConf), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func denyReason(err error) string {
	var de *DeniedError
	if errors.As(err, &de) {
		return de.Reason
	}
	return ""
}

// A tunnel brought up by an older Kongtrol (UAC /installtunnelservice) or
// the WireGuard app is adopted with the caller's own profile config.
func TestWGRemove_AdoptsPreexistingTunnel(t *testing.T) {
	s := stubWG(t, func(n string) (bool, error) { return n == "rluna", nil })
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	ctx := context.Background()
	conf := profileConf(t, "rluna")

	if err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna", ConfigPath: conf}); err != nil {
		t.Fatalf("adopt+remove: %v", err)
	}
	if len(s.downs) != 1 || s.downs[0] != "rluna" {
		t.Fatalf("downs = %v", s.downs)
	}
	if l.wgManaged("rluna") {
		t.Fatal("private copy left after remove")
	}
	// Already gone: nothing to do, no error.
	s2 := stubWG(t, func(string) (bool, error) { return false, nil })
	if err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna", ConfigPath: conf}); err != nil || len(s2.downs) != 0 {
		t.Fatalf("absent tunnel: err=%v downs=%v", err, s2.downs)
	}
}

func TestWGRemove_AdoptionRules(t *testing.T) {
	ctx := context.Background()
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})

	s := stubWG(t, func(string) (bool, error) { return true, nil })
	err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna"})
	if denyReason(err) != DenyWGNotManaged {
		t.Fatalf("no config: %v", err)
	}
	if k, ok := GuidanceKey(err); !ok || k != "privops.denied.wg_not_managed" {
		t.Fatalf("guidance key %q", k)
	}
	err = l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna", ConfigPath: profileConf(t, "other")})
	if denyReason(err) != DenyWGNotManaged {
		t.Fatalf("name mismatch: %v", err)
	}
	if len(s.downs) != 0 {
		t.Fatalf("removed without authorization: %v", s.downs)
	}

	// A service/interface of that name that is not WireGuard is never removed.
	s = stubWG(t, func(n string) (bool, error) {
		return false, denied(DenyNotWireGuard, "service runs something else")
	})
	err = l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "spooler", ConfigPath: profileConf(t, "spooler")})
	if denyReason(err) != DenyNotWireGuard || len(s.downs) != 0 {
		t.Fatalf("non-WireGuard service: err=%v downs=%v", err, s.downs)
	}

	// Config that is not a WireGuard config is rejected before any check.
	bad := filepath.Join(t.TempDir(), "rluna.conf")
	_ = os.WriteFile(bad, []byte("[Interface]\nPostUp = calc.exe\n"), 0o600)
	if err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna", ConfigPath: bad}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("hook config: %v", err)
	}
}

func TestWGSetAllowedIPs_Adopts(t *testing.T) {
	testWGSetAllowedIPsAdopts(t, "us-content-1") // valid on every OS (Linux IFNAMSIZ = 15)
}

// testWGSetAllowedIPsAdopts checks that allowed-IPs can be set on a tunnel Kongtrol
// did not start (adoption). Windows-only names longer than 15 chars are covered in
// wgadopt_windows_test.go.
func testWGSetAllowedIPsAdopts(t *testing.T, name string) {
	t.Helper()
	s := stubWG(t, func(n string) (bool, error) { return n == name, nil })
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	req := WGAllowedIPsRequest{Interface: name, PeerPublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
		AllowedIPs: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, ConfigPath: profileConf(t, name)}
	if err := l.WGSetAllowedIPs(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(s.sets) != 1 || !l.wgManaged(req.Interface) {
		t.Fatalf("sets=%v managed=%v", s.sets, l.wgManaged(req.Interface))
	}
}

// Denial reasons survive the IPC round trip, so clients show the right line.
func TestDeniedReason_OverTheWire(t *testing.T) {
	f := &fakeOps{err: denied(DenyFileWritable, "x.conf can be modified by other users")}
	err := pipeClient(f).WGRemoveTunnel(context.Background(), WGRemoveRequest{Name: "wg0"})
	if !errors.Is(err, ErrDenied) || denyReason(err) != DenyFileWritable {
		t.Fatalf("err = %#v", err)
	}
	if k, _ := GuidanceKey(err); k != "privops.denied.file_writable" {
		t.Fatalf("key %q", k)
	}
	// Untyped denial (caller not allowed) keeps the generic guidance.
	f.err = ErrDenied
	err = pipeClient(f).KillSwitchDisable(context.Background())
	if k, _ := GuidanceKey(err); k != "service.error.denied" {
		t.Fatalf("generic key %q (%v)", k, err)
	}
}
