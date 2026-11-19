package sockowner

import (
	"errors"
	"net"
	"net/netip"
	"runtime"
	"testing"
)

func ap(s string) netip.AddrPort { return netip.MustParseAddrPort(s) }

func TestMatchPrefersTheConnectionRow(t *testing.T) {
	client, server := ap("127.0.0.1:50000"), ap("127.0.0.1:9741")
	rows := []row{
		{Local: ap("0.0.0.0:9741"), Listen: true, PID: 1},
		{Local: ap("127.0.0.1:9741"), Listen: true, PID: 2},
		{Local: client, Remote: server, PID: 3}, // our own end
		{Local: server, Remote: client, PID: 4}, // the server end
	}
	if r, ok := match(rows, client, server); !ok || r.PID != 4 {
		t.Fatalf("connection row: %+v %v", r, ok)
	}
	if r, ok := match(rows[:2], client, server); !ok || r.PID != 2 {
		t.Fatalf("exact LISTEN row first: %+v %v", r, ok)
	}
	if r, ok := match(rows[:1], client, server); !ok || r.PID != 1 {
		t.Fatalf("wildcard LISTEN row: %+v %v", r, ok)
	}
	if _, ok := match(rows[2:3], client, server); ok {
		t.Fatal("our own end must not count as the server")
	}
}

// A real loopback connection to a listener of this very process.
func TestCheckRealLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			defer c.Close()
			var b [1]byte
			_, _ = c.Read(b[:])
		}
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := Check(c.LocalAddr().(*net.TCPAddr).AddrPort(), c.RemoteAddr().(*net.TCPAddr).AddrPort())
	switch runtime.GOOS {
	case "windows", "linux":
		if err != nil || res.Verdict != Self {
			t.Fatalf("own listener: %+v %v", res, err)
		}
	default:
		if res.Verdict != Unknown || !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unsupported OS: %+v %v", res, err)
		}
	}
}
