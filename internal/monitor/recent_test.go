package monitor

import (
	"context"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type recentFake struct {
	mu    sync.Mutex
	conns []TCPConn
	now   time.Time
}

func (f *recentFake) set(c ...TCPConn) {
	f.mu.Lock()
	f.conns = c
	f.mu.Unlock()
}

func (f *recentFake) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func newRecentForTest(f *recentFake) *RecentConnections {
	r := NewRecentConnections(true)
	r.supported = true
	r.now = func() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.now }
	r.conns = func() ([]TCPConn, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		return append([]TCPConn(nil), f.conns...), nil
	}
	r.names = func(cs []TCPConn) map[TCPConn]string {
		out := map[TCPConn]string{}
		for _, c := range cs {
			if c.PID == 42 {
				out[c] = "chrome.exe"
			}
		}
		return out
	}
	r.ifaces = func() map[netip.Addr]string {
		return map[netip.Addr]string{
			netip.MustParseAddr("192.168.1.10"): "Wi-Fi",
			netip.MustParseAddr("10.8.0.2"):     "wg-office",
		}
	}
	r.Via = func(a netip.Addr) string {
		if a == netip.MustParseAddr("10.8.0.2") {
			return "office"
		}
		return ""
	}
	r.Host = func(a netip.Addr) string {
		if a == netip.MustParseAddr("10.0.0.5") {
			return "intranet.example"
		}
		return ""
	}
	return r
}

func conn(local, remote string, pid uint32) TCPConn {
	return TCPConn{Local: netip.MustParseAddrPort(local), Remote: netip.MustParseAddrPort(remote), PID: pid}
}

func TestRecentConnections_ListsDestinations(t *testing.T) {
	f := &recentFake{now: time.Unix(1_700_000_000, 0)}
	r := newRecentForTest(f)
	f.set(
		conn("192.168.1.10:50000", "93.184.216.34:443", 42),
		conn("192.168.1.10:50001", "93.184.216.34:443", 42), // 2nd connection, same destination
		conn("10.8.0.2:40000", "10.0.0.5:22", 0),
		conn("127.0.0.1:1", "127.0.0.1:9741", 0), // loopback: never listed
	)
	r.Check()
	f.advance(15 * time.Second)
	r.Check() // the same two connections still open: not counted again

	got := r.List(0)
	if len(got) != 2 {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	byIP := map[string]RecentConnection{}
	for _, c := range got {
		byIP[c.RemoteIP] = c
	}
	web := byIP["93.184.216.34"]
	if web.Port != 443 || web.Process != "chrome.exe" || web.Via != "" || web.Interface != "Wi-Fi" || web.Count != 2 {
		t.Fatalf("web = %+v", web)
	}
	ssh := byIP["10.0.0.5"]
	if ssh.Host != "intranet.example" || ssh.Via != "office" || ssh.Interface != "wg-office" || ssh.Count != 1 {
		t.Fatalf("ssh = %+v", ssh)
	}
	if !web.LastSeen.After(web.FirstSeen) {
		t.Fatalf("last seen not updated: %+v", web)
	}

	// The connection closes and a new one opens later: counted.
	f.set()
	f.advance(15 * time.Second)
	r.Check()
	f.set(conn("192.168.1.10:50000", "93.184.216.34:443", 42))
	f.advance(15 * time.Second)
	r.Check()
	if c := r.List(1)[0]; c.RemoteIP != "93.184.216.34" || c.Count != 3 {
		t.Fatalf("most recent first, new connection counted: %+v", c)
	}
}

func TestRecentConnections_BoundAndExpiry(t *testing.T) {
	f := &recentFake{now: time.Unix(1_700_000_000, 0)}
	r := newRecentForTest(f)
	r.max = 3
	for i := range 5 {
		f.set(conn("192.168.1.10:50000", netip.AddrPortFrom(netip.AddrFrom4([4]byte{1, 1, 1, byte(i)}), 443).String(), 0))
		f.advance(time.Second)
		r.Check()
	}
	got := r.List(0)
	if len(got) != 3 || got[0].RemoteIP != "1.1.1.4" || got[2].RemoteIP != "1.1.1.2" {
		t.Fatalf("bounded list keeps the newest: %+v", got)
	}
	f.advance(RecentTTL + time.Minute)
	if got := r.List(0); len(got) != 0 {
		t.Fatalf("entries unseen for more than an hour must go: %+v", got)
	}
}

func TestRecentConnections_OffForgetsAndReadsNothing(t *testing.T) {
	f := &recentFake{now: time.Unix(1_700_000_000, 0)}
	r := newRecentForTest(f)
	f.set(conn("192.168.1.10:50000", "8.8.8.8:53", 0))
	r.Check()
	r.SetEnabled(false)
	if len(r.List(0)) != 0 {
		t.Fatal("turning it off must erase the list")
	}
	reads := 0
	r.conns = func() ([]TCPConn, error) { reads++; return nil, nil }
	r.Check()
	if reads != 0 {
		t.Fatal("nothing is read while off")
	}

	// Shutdown erases too (memory only: nothing to keep).
	r.SetEnabled(true)
	r.conns = func() ([]TCPConn, error) { return []TCPConn{conn("192.168.1.10:1", "8.8.4.4:53", 0)}, nil }
	r.Check()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.Run(ctx, time.Hour)
	if len(r.List(0)) != 0 {
		t.Fatal("Run must forget on shutdown")
	}
}

// TestRecentConnections_MemoryOnly guards the privacy rule: the type has no
// path, file, writer or logger to persist or leak entries through.
func TestRecentConnections_MemoryOnly(t *testing.T) {
	typ := reflect.TypeOf(RecentConnections{})
	for i := range typ.NumField() {
		f := typ.Field(i)
		name := strings.ToLower(f.Name)
		kind := strings.ToLower(f.Type.String())
		for _, bad := range []string{"path", "file", "log", "writer", "zap"} {
			if strings.Contains(name, bad) || strings.Contains(kind, bad) {
				t.Fatalf("field %s %s could persist or log destinations", f.Name, f.Type)
			}
		}
	}
}

func TestParseProcNetTCPConns(t *testing.T) {
	text := "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n" +
		"   0: 0A00A8C0:C350 22D8B85D:01BB 01 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 20 4 30 10 -1\n" +
		"   1: 0100007F:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 999 1\n"
	got := parseProcNetTCPConns(text)
	want := []TCPConn{{
		Local:  netip.MustParseAddrPort("192.168.0.10:50000"),
		Remote: netip.MustParseAddrPort("93.184.216.34:443"),
		Inode:  12345,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
