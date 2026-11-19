//go:build windows

package monitor

import (
	"encoding/binary"
	"net/netip"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetExtendedTcpTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

const (
	tcpTableOwnerPIDAll = 5
	tcpStateSynSent     = 3
	tcpStateEstablished = 5
)

// tcpConnectionsSupported: GetExtendedTcpTable needs no elevation.
const tcpConnectionsSupported = true

// activeRemoteAddrs returns the remote addresses of TCP connections that
// are established or being opened (GetExtendedTcpTable, no elevation).
func activeRemoteAddrs() ([]netip.Addr, error) {
	conns, err := tcpConnections()
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(conns))
	for _, c := range conns {
		out = append(out, c.Remote.Addr())
	}
	return out, nil
}

// tcpConnections lists established / opening TCP connections with their
// owning process id (MIB_TCPROW_OWNER_PID / MIB_TCP6ROW_OWNER_PID).
func tcpConnections() ([]TCPConn, error) {
	var out []TCPConn
	for _, af := range []uint32{windows.AF_INET, windows.AF_INET6} {
		buf, err := tcpTable(af)
		if err != nil {
			return nil, err
		}
		out = append(out, parseTCPTable(buf, af == windows.AF_INET6)...)
	}
	return out, nil
}

// parseTCPTable reads a TCP_TABLE_OWNER_PID_ALL buffer. Row layouts:
// IPv4 (24 bytes) state, localAddr, localPort, remoteAddr, remotePort,
// pid; IPv6 (56 bytes) localAddr[16], localScope, localPort,
// remoteAddr[16], remoteScope, remotePort, state, pid. Ports are in
// network byte order in the low 16 bits.
func parseTCPTable(buf []byte, v6 bool) []TCPConn {
	if len(buf) < 4 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(buf))
	rowSize := 24
	if v6 {
		rowSize = 56
	}
	var out []TCPConn
	for i := 0; i < n; i++ {
		row := 4 + i*rowSize
		if row+rowSize > len(buf) {
			break
		}
		r := buf[row : row+rowSize]
		var c TCPConn
		var state uint32
		if v6 {
			state = binary.LittleEndian.Uint32(r[48:])
			c.Local = netip.AddrPortFrom(netip.AddrFrom16([16]byte(r[0:16])).Unmap(), binary.BigEndian.Uint16(r[20:22]))
			c.Remote = netip.AddrPortFrom(netip.AddrFrom16([16]byte(r[24:40])).Unmap(), binary.BigEndian.Uint16(r[44:46]))
			c.PID = binary.LittleEndian.Uint32(r[52:])
		} else {
			state = binary.LittleEndian.Uint32(r[0:])
			c.Local = netip.AddrPortFrom(netip.AddrFrom4([4]byte(r[4:8])), binary.BigEndian.Uint16(r[8:10]))
			c.Remote = netip.AddrPortFrom(netip.AddrFrom4([4]byte(r[12:16])), binary.BigEndian.Uint16(r[16:18]))
			c.PID = binary.LittleEndian.Uint32(r[20:])
		}
		if state != tcpStateEstablished && state != tcpStateSynSent {
			continue
		}
		out = append(out, c)
	}
	return out
}

func tcpTable(af uint32) ([]byte, error) {
	size := uint32(16 << 10)
	for range 4 {
		buf := make([]byte, size)
		r, _, _ := procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)),
			0, uintptr(af), tcpTableOwnerPIDAll, 0)
		switch windows.Errno(r) {
		case 0:
			return buf[:size], nil
		case windows.ERROR_INSUFFICIENT_BUFFER:
			size += 4 << 10
			continue
		default:
			return nil, windows.Errno(r)
		}
	}
	return nil, windows.ERROR_INSUFFICIENT_BUFFER
}

// processNames maps conns to the executable name of their owning process
// (OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION, which a standard
// user may use on most processes; protected and other users' system
// processes stay unnamed).
func processNames(conns []TCPConn) map[TCPConn]string {
	byPID := map[uint32]string{}
	names := map[TCPConn]string{}
	for _, c := range conns {
		if c.PID == 0 {
			continue
		}
		n, ok := byPID[c.PID]
		if !ok {
			n = exeName(c.PID)
			byPID[c.PID] = n
		}
		if n != "" {
			names[c] = n
		}
	}
	return names
}

func exeName(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:size]))
}
