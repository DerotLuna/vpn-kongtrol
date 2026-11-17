//go:build windows

package monitor

import (
	"encoding/binary"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetExtendedTcpTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

const (
	tcpTableOwnerPIDAll = 5
	tcpStateSynSent     = 3
	tcpStateEstablished = 5
)

// activeRemoteAddrs returns the remote addresses of TCP connections that
// are established or being opened (GetExtendedTcpTable, no elevation).
func activeRemoteAddrs() ([]netip.Addr, error) {
	var out []netip.Addr
	for _, af := range []uint32{windows.AF_INET, windows.AF_INET6} {
		buf, err := tcpTable(af)
		if err != nil {
			return nil, err
		}
		if len(buf) < 4 {
			continue
		}
		n := int(binary.LittleEndian.Uint32(buf))
		rowSize, remoteOff, stateOff := 24, 12, 0
		if af == windows.AF_INET6 {
			rowSize, remoteOff, stateOff = 56, 24, 48
		}
		for i := 0; i < n; i++ {
			row := 4 + i*rowSize
			if row+rowSize > len(buf) {
				break
			}
			state := binary.LittleEndian.Uint32(buf[row+stateOff:])
			if state != tcpStateEstablished && state != tcpStateSynSent {
				continue
			}
			var a netip.Addr
			if af == windows.AF_INET {
				a = netip.AddrFrom4([4]byte(buf[row+remoteOff : row+remoteOff+4]))
			} else {
				a = netip.AddrFrom16([16]byte(buf[row+remoteOff : row+remoteOff+16])).Unmap()
			}
			out = append(out, a)
		}
	}
	return out, nil
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
