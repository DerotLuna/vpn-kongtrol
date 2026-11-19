//go:build windows

package monitor

import (
	"encoding/binary"
	"net/netip"
	"testing"
)

func TestParseTCPTable_IPv4AndIPv6(t *testing.T) {
	v4 := make([]byte, 4+2*24)
	binary.LittleEndian.PutUint32(v4, 2)
	row := v4[4:28]
	binary.LittleEndian.PutUint32(row[0:], tcpStateEstablished)
	copy(row[4:8], []byte{192, 168, 1, 10})
	binary.BigEndian.PutUint16(row[8:10], 50000)
	copy(row[12:16], []byte{93, 184, 216, 34})
	binary.BigEndian.PutUint16(row[16:18], 443)
	binary.LittleEndian.PutUint32(row[20:], 4242)
	binary.LittleEndian.PutUint32(v4[28:], 2) // LISTEN: skipped

	got := parseTCPTable(v4, false)
	want := TCPConn{
		Local:  netip.MustParseAddrPort("192.168.1.10:50000"),
		Remote: netip.MustParseAddrPort("93.184.216.34:443"),
		PID:    4242,
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("v4 got %+v", got)
	}

	v6 := make([]byte, 4+56)
	binary.LittleEndian.PutUint32(v6, 1)
	r := v6[4:]
	local := netip.MustParseAddr("2001:db8::1").As16()
	remote := netip.MustParseAddr("2606:4700::1111").As16()
	copy(r[0:16], local[:])
	binary.BigEndian.PutUint16(r[20:22], 40000)
	copy(r[24:40], remote[:])
	binary.BigEndian.PutUint16(r[44:46], 443)
	binary.LittleEndian.PutUint32(r[48:], tcpStateSynSent)
	binary.LittleEndian.PutUint32(r[52:], 7)
	got = parseTCPTable(v6, true)
	if len(got) != 1 || got[0].Remote != netip.MustParseAddrPort("[2606:4700::1111]:443") ||
		got[0].Local.Port() != 40000 || got[0].PID != 7 {
		t.Fatalf("v6 got %+v", got)
	}
}

// TestTCPConnections_NoAdmin reads the real table as this (standard) user.
func TestTCPConnections_NoAdmin(t *testing.T) {
	conns, err := tcpConnections()
	if err != nil {
		t.Fatalf("GetExtendedTcpTable without admin: %v", err)
	}
	names := processNames(conns) // best effort; must not fail or block
	t.Logf("%d connections, %d with a program name", len(conns), len(names))
}
