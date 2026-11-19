//go:build windows

package sockowner

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// v4Table builds a TCP_TABLE_OWNER_PID_ALL buffer with one LISTEN row.
func v4Table(port uint16, pid uint32) []byte {
	b := make([]byte, 4+24)
	binary.LittleEndian.PutUint32(b, 1)
	r := b[4:]
	binary.LittleEndian.PutUint32(r[0:], tcpStateListen)
	copy(r[4:8], []byte{127, 0, 0, 1})
	binary.BigEndian.PutUint16(r[8:10], port)
	binary.LittleEndian.PutUint32(r[20:], pid)
	return b
}

// v6Table builds an IPv6 table with one row (local [addr]:port).
func v6Table(addr string, port uint16, state, pid uint32) []byte {
	b := make([]byte, 4+56)
	binary.LittleEndian.PutUint32(b, 1)
	r := b[4:]
	a := netip.MustParseAddr(addr).As16()
	copy(r[0:16], a[:])
	binary.BigEndian.PutUint16(r[20:22], port)
	binary.LittleEndian.PutUint32(r[48:], state)
	binary.LittleEndian.PutUint32(r[52:], pid)
	return b
}

func fakeWindows(t *testing.T, v4, v6 []byte, info procInfo, infoErr error) {
	t.Helper()
	oldT, oldU, oldS, oldP := readTable, pidUser, selfUser, selfPID
	t.Cleanup(func() { readTable, pidUser, selfUser, selfPID = oldT, oldU, oldS, oldP })
	readTable = func(af uint32) ([]byte, error) {
		if af == windows.AF_INET6 {
			if v6 == nil {
				return nil, errors.New("no v6")
			}
			return v6, nil
		}
		if v4 == nil {
			return nil, errors.New("no v4")
		}
		return v4, nil
	}
	pidUser = func(uint32) (procInfo, error) { return info, infoErr }
	selfUser = func() (string, error) { return "S-1-5-21-1-1-1-1001", nil }
	selfPID = func() uint32 { return 1 }
}

func TestCheckWindowsVerdicts(t *testing.T) {
	client, server := ap("127.0.0.1:50000"), ap("127.0.0.1:9741")
	created := time.Unix(1_700_000_000, 0)
	cases := []struct {
		name   string
		info   procInfo
		err    error
		want   Verdict
		opened bool
	}{
		{"same user", procInfo{sid: "S-1-5-21-1-1-1-1001", opened: true}, nil, Self, false},
		{"another user", procInfo{sid: "S-1-5-21-1-1-1-1002", opened: true}, nil, Other, false},
		{"process not openable", procInfo{}, windows.ERROR_ACCESS_DENIED, Denied, false},
		{"only the token denied", procInfo{opened: true, created: created}, windows.ERROR_ACCESS_DENIED, Denied, true},
	}
	for _, tc := range cases {
		fakeWindows(t, v4Table(9741, 4242), nil, tc.info, tc.err)
		res, err := Check(client, server)
		if err != nil || res.Verdict != tc.want || res.PID != 4242 || res.ProcessOpened != tc.opened {
			t.Errorf("%s: %+v %v, want %v opened=%v", tc.name, res, err, tc.want, tc.opened)
		}
		if tc.opened && !res.Created.Equal(created) {
			t.Errorf("%s: created=%v", tc.name, res.Created)
		}
	}
	fakeWindows(t, v4Table(9999, 4242), nil, procInfo{}, nil)
	if res, err := Check(client, server); res.Verdict != Unknown || err == nil {
		t.Errorf("no row: %+v %v", res, err)
	}
}

// A dual-mode [::] listener appears only in the IPv6 table; the check must
// still find it for a 127.0.0.1 connection (not Unknown = fail-open).
func TestCheckWindowsDualModeListener(t *testing.T) {
	client, server := ap("127.0.0.1:50000"), ap("127.0.0.1:9741")
	fakeWindows(t, make([]byte, 4), v6Table("::", 9741, tcpStateListen, 77),
		procInfo{sid: "S-1-5-21-1-1-1-1002", opened: true}, nil)
	if res, err := Check(client, server); err != nil || res.Verdict != Other || res.PID != 77 {
		t.Fatalf("dual-mode listener of another user: %+v %v", res, err)
	}
	// The server end of the connection itself, as an IPv4-mapped row.
	conn := make([]byte, 4+56)
	binary.LittleEndian.PutUint32(conn, 1)
	r := conn[4:]
	l, rm := netip.MustParseAddr("::ffff:127.0.0.1").As16(), netip.MustParseAddr("::ffff:127.0.0.1").As16()
	copy(r[0:16], l[:])
	binary.BigEndian.PutUint16(r[20:22], 9741)
	copy(r[24:40], rm[:])
	binary.BigEndian.PutUint16(r[44:46], 50000)
	binary.LittleEndian.PutUint32(r[48:], 5) // established
	binary.LittleEndian.PutUint32(r[52:], 88)
	fakeWindows(t, nil, conn, procInfo{sid: "S-1-5-21-1-1-1-1001", opened: true}, nil)
	if res, err := Check(client, server); err != nil || res.Verdict != Self || res.PID != 88 {
		t.Fatalf("mapped connection row (v4 table unreadable): %+v %v", res, err)
	}
}
