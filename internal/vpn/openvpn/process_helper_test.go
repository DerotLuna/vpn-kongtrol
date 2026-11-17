package openvpn

import (
	"bufio"
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// fakeMgmt emulates openvpn's management interface with a password file:
// "ENTER PASSWORD:" (no newline), then the banner, then answers commands.
type fakeMgmt struct {
	ln       net.Listener
	password string
	mu       sync.Mutex
	cmds     []string
}

func newFakeMgmt(t *testing.T, password string) *fakeMgmt {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeMgmt{ln: ln, password: password}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	return f
}

func (f *fakeMgmt) serve(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	if f.password != "" {
		_, _ = c.Write([]byte("ENTER PASSWORD:"))
		line, _ := r.ReadString('\n')
		if strings.TrimSpace(line) != f.password {
			_, _ = c.Write([]byte("ERROR: bad password\r\n"))
			return
		}
		_, _ = c.Write([]byte("SUCCESS: password is correct\r\n"))
	}
	_, _ = c.Write([]byte(">INFO:OpenVPN Management Interface Version 5 -- type 'help' for more info\r\n"))
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(line)
		f.mu.Lock()
		f.cmds = append(f.cmds, cmd)
		f.mu.Unlock()
		switch {
		case cmd == "state":
			_, _ = c.Write([]byte(">PASSWORD:Need 'Auth' username/password\r\n1720000000,CONNECTED,SUCCESS,10.8.0.6,1.2.3.4,1194,,\r\nEND\r\n"))
		case strings.HasPrefix(cmd, "username"), strings.HasPrefix(cmd, "password"), strings.HasPrefix(cmd, "signal"):
			_, _ = c.Write([]byte("SUCCESS: ok\r\n"))
		}
	}
}

func (f *fakeMgmt) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.cmds...)
}

// fakeHelper is a privops.Ops whose OpenVPN session points at fakeMgmt.
type fakeHelper struct {
	privops.Ops
	addr    string
	mu      sync.Mutex
	req     privops.OpenVPNStartRequest
	running bool
	stopped bool
}

func (h *fakeHelper) OpenVPNStart(_ context.Context, r privops.OpenVPNStartRequest) (privops.OpenVPNSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.req, h.running = r, true
	return privops.OpenVPNSession{ID: strings.Repeat("a", 32), MgmtAddr: h.addr, MgmtPassword: "s3cret"}, nil
}

func (h *fakeHelper) OpenVPNStatus(context.Context, string) (privops.OpenVPNState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return privops.OpenVPNState{Running: h.running, ExitError: map[bool]string{false: "exit status 1"}[h.running], LogTail: []string{
		"TUN/TAP device tun7 opened",
		"PUSH: Received control message: 'PUSH_REPLY,route 10.0.0.0 255.0.0.0,dhcp-option DNS 10.0.0.53'",
		"Initialization Sequence Completed",
	}}, nil
}

func (h *fakeHelper) OpenVPNStop(context.Context, string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running, h.stopped = false, true
	return nil
}

func useFakeHelper(t *testing.T, h privops.Ops, err error) {
	oldOps, oldElev, oldPoll := opsFunc, isElevated, helperPollEvery
	t.Cleanup(func() { opsFunc, isElevated, helperPollEvery = oldOps, oldElev, oldPoll })
	opsFunc = func(context.Context) (privops.Ops, error) { return h, err }
	isElevated = func() bool { return false }
	helperPollEvery = 20 * time.Millisecond
}

func TestStartViaHelper_AuthLogAndStop(t *testing.T) {
	m := newFakeMgmt(t, "s3cret")
	h := &fakeHelper{addr: m.ln.Addr().String()}
	useFakeHelper(t, h, nil)

	cfg := filepath.Join("profiles", "office.ovpn")
	p, err := start(context.Background(), startOptions{ConfigPath: cfg}, "alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := filepath.Abs(cfg); h.req.ConfigPath != want {
		t.Fatalf("helper got config %q, want absolute %q", h.req.ConfigPath, want)
	}
	st, err := p.mgmt.State()
	if err != nil || !strings.HasPrefix(st, "1720000000,CONNECTED") {
		t.Fatalf("state %q err %v (notifications/SUCCESS lines must be skipped)", st, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if iface, dns := p.log.snapshot(); iface == "tun7" && len(dns) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if iface, dns := p.log.snapshot(); iface != "tun7" || len(dns) != 1 || dns[0].String() != "10.0.0.53" {
		t.Fatalf("log not mirrored from helper: iface=%q dns=%v", iface, dns)
	}
	if p.hasExited() {
		t.Fatal("running session reported as exited")
	}
	if err := p.stop(); err != nil {
		t.Fatal(err)
	}
	if !h.stopped || !p.hasExited() {
		t.Fatal("stop did not end the helper session")
	}
	// The fake management server records commands asynchronously; on slow CI
	// runners "signal SIGTERM" can land a moment after stop() returns.
	var cmds string
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		cmds = strings.Join(m.sent(), "\n")
		if strings.Contains(cmds, `username "Auth" alice`) && strings.Contains(cmds, "signal SIGTERM") {
			break
		}
	}
	if !strings.Contains(cmds, `username "Auth" alice`) || !strings.Contains(cmds, "signal SIGTERM") {
		t.Fatalf("management commands: %s", cmds)
	}
}

func TestStartViaHelper_DetectsExit(t *testing.T) {
	m := newFakeMgmt(t, "s3cret")
	h := &fakeHelper{addr: m.ln.Addr().String()}
	useFakeHelper(t, h, nil)
	p, err := start(context.Background(), startOptions{ConfigPath: "x.ovpn"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	h.running = false
	h.mu.Unlock()
	select {
	case <-p.exited:
	case <-time.After(2 * time.Second):
		t.Fatal("helper-side exit not noticed")
	}
	if !strings.Contains(p.exitError().Error(), "exit status 1") {
		t.Fatalf("exit error %v", p.exitError())
	}
}

func TestStartViaHelper_NotInstalled(t *testing.T) {
	useFakeHelper(t, nil, privops.ErrHelperNotInstalled)
	_, err := start(context.Background(), startOptions{ConfigPath: "x.ovpn"}, "", "")
	if !errors.Is(err, privops.ErrHelperNotInstalled) {
		t.Fatalf("err = %v", err)
	}
	if k, ok := privops.GuidanceKey(err); !ok || k != "service.error.not_installed" {
		t.Fatalf("guidance key %q", k)
	}
}

func TestMgmtClient_WrongPassword(t *testing.T) {
	m := newFakeMgmt(t, "right")
	if _, err := newMgmtClientAuth(m.ln.Addr().String(), "wrong"); err == nil {
		t.Fatal("wrong management password accepted")
	}
	if _, err := newMgmtClientAuth(m.ln.Addr().String(), ""); err == nil {
		t.Fatal("missing management password accepted")
	}
}
