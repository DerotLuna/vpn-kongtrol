//go:build windows

package sockowner

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"os"
	"time"
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
	// Both tables, merged (IPv4-mapped addresses unmapped): a dual-mode
	// [::] listener answers 127.0.0.1 from the IPv6 table, and reading only
	// one would end in Unknown (fail-open).
	var rows []row
	var firstErr error
	for _, af := range []uint32{windows.AF_INET, windows.AF_INET6} {
		buf, err := readTable(af)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rows = append(rows, parseTable(buf, af == windows.AF_INET6)...)
	}
	if len(rows) == 0 && firstErr != nil {
		return Result{}, firstErr
	}
	r, ok := match(rows, local, remote)
	if !ok || r.PID == 0 {
		return Result{}, ErrNotFound
	}
	res := Result{PID: r.PID}
	if r.PID == selfPID() {
		res.Verdict = Self
		return res, nil
	}
	info, err := pidUser(r.PID)
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			// Another user's process, an elevated one or a service. The
			// caller decides with ProcessOpened and Created (ADR-0062).
			res.Verdict = Denied
			res.ProcessOpened, res.Created = info.opened, info.created
			return res, nil
		}
		return Result{}, err
	}
	owner := info.sid
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

// procInfo is what processUserSID learned about a process.
type procInfo struct {
	sid     string    // user SID of its token ("" when the token was denied)
	opened  bool      // OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION) worked
	created time.Time // creation time (GetProcessTimes), when opened
}

// processUserSID reads the user SID of process pid's token
// (PROCESS_QUERY_LIMITED_INFORMATION + TOKEN_QUERY, no admin needed for
// this user's own processes). When the token is denied but the process
// could be opened, the error is ERROR_ACCESS_DENIED and info still says
// opened and when the process was created.
func processUserSID(pid uint32) (procInfo, error) {
	var info procInfo
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return info, err
	}
	defer windows.CloseHandle(h)
	info.opened = true
	var creation, exit, kernel, user windows.Filetime
	if windows.GetProcessTimes(h, &creation, &exit, &kernel, &user) == nil {
		info.created = time.Unix(0, creation.Nanoseconds())
	}
	var tok windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok); err != nil {
		return info, err
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return info, err
	}
	info.sid = u.User.Sid.String()
	return info, nil
}

func currentUserSID() (string, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return u.User.Sid.String(), nil
}
