//go:build windows

package sockowner

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetExtendedTcpTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

const (
	tcpTableOwnerPIDAll = 5
	tcpStateListen      = 2
)

// Test seams.
var (
	readTable = tcpTable
	pidUser   = processUserSID
	selfUser  = currentUserSID
	selfPID   = func() uint32 { return uint32(os.Getpid()) }
)

func check(local, remote netip.AddrPort) (Result, error) {
	af := uint32(windows.AF_INET)
	if remote.Addr().Is6() {
		af = windows.AF_INET6
	}
	buf, err := readTable(af)
	if err != nil {
		return Result{}, err
	}
	r, ok := match(parseTable(buf, af == windows.AF_INET6), local, remote)
	if !ok || r.PID == 0 {
		return Result{}, ErrNotFound
	}
	res := Result{PID: r.PID}
	if r.PID == selfPID() {
		res.Verdict = Self
		return res, nil
	}
	owner, err := pidUser(r.PID)
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			// Another user's process, an elevated one or a service: the
			// caller decides (its own daemon's PID file, ADR-0062).
			res.Verdict = Denied
			return res, nil
		}
		return Result{}, err
	}
	me, err := selfUser()
	if err != nil {
		return Result{}, err
	}
	res.Owner = owner
	if owner == me {
		res.Verdict = Self
	} else {
		res.Verdict = Other
	}
	return res, nil
}

// parseTable reads a TCP_TABLE_OWNER_PID_ALL buffer, every state. Row
// layouts: IPv4 (24 bytes) state, localAddr, localPort, remoteAddr,
// remotePort, pid; IPv6 (56 bytes) localAddr[16], localScope, localPort,
// remoteAddr[16], remoteScope, remotePort, state, pid. Ports are in network
// byte order in the low 16 bits.
func parseTable(buf []byte, v6 bool) []row {
	if len(buf) < 4 {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(buf))
	size := 24
	if v6 {
		size = 56
	}
	out := make([]row, 0, n)
	for i := 0; i < n; i++ {
		off := 4 + i*size
		if off+size > len(buf) {
			break
		}
		b := buf[off : off+size]
		var r row
		var state uint32
		if v6 {
			r.Local = netip.AddrPortFrom(netip.AddrFrom16([16]byte(b[0:16])).Unmap(), binary.BigEndian.Uint16(b[20:22]))
			r.Remote = netip.AddrPortFrom(netip.AddrFrom16([16]byte(b[24:40])).Unmap(), binary.BigEndian.Uint16(b[44:46]))
			state = binary.LittleEndian.Uint32(b[48:])
			r.PID = binary.LittleEndian.Uint32(b[52:])
		} else {
			state = binary.LittleEndian.Uint32(b[0:])
			r.Local = netip.AddrPortFrom(netip.AddrFrom4([4]byte(b[4:8])), binary.BigEndian.Uint16(b[8:10]))
			r.Remote = netip.AddrPortFrom(netip.AddrFrom4([4]byte(b[12:16])), binary.BigEndian.Uint16(b[16:18]))
			r.PID = binary.LittleEndian.Uint32(b[20:])
		}
		r.Listen = state == tcpStateListen
		r.UID = -1
		out = append(out, r)
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

// processUserSID is the user SID of process pid's token
// (PROCESS_QUERY_LIMITED_INFORMATION + TOKEN_QUERY, no admin needed for
// this user's own processes).
func processUserSID(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	var tok windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok); err != nil {
		return "", err
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return "", err
	}
	return u.User.Sid.String(), nil
}

func currentUserSID() (string, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return u.User.Sid.String(), nil
}
