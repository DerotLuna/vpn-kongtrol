//go:build windows

package sockowner

import (
	"encoding/binary"
	"testing"

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

func fakeWindows(t *testing.T, table []byte, owner string, ownerErr error) {
	t.Helper()
	oldT, oldU, oldS, oldP := readTable, pidUser, selfUser, selfPID
	t.Cleanup(func() { readTable, pidUser, selfUser, selfPID = oldT, oldU, oldS, oldP })
	readTable = func(uint32) ([]byte, error) { return table, nil }
	pidUser = func(uint32) (string, error) { return owner, ownerErr }
	selfUser = func() (string, error) { return "S-1-5-21-1-1-1-1001", nil }
	selfPID = func() uint32 { return 1 }
}

func TestCheckWindowsVerdicts(t *testing.T) {
	client, server := ap("127.0.0.1:50000"), ap("127.0.0.1:9741")
	cases := []struct {
		name  string
		owner string
		err   error
		want  Verdict
	}{
		{"same user", "S-1-5-21-1-1-1-1001", nil, Self},
		{"another user", "S-1-5-21-1-1-1-1002", nil, Other},
		{"cannot open", "", windows.ERROR_ACCESS_DENIED, Denied},
	}
	for _, tc := range cases {
		fakeWindows(t, v4Table(9741, 4242), tc.owner, tc.err)
		res, err := Check(client, server)
		if err != nil || res.Verdict != tc.want || res.PID != 4242 {
			t.Errorf("%s: %+v %v, want %v", tc.name, res, err, tc.want)
		}
	}
	fakeWindows(t, v4Table(9999, 4242), "", nil)
	if res, err := Check(client, server); res.Verdict != Unknown || err == nil {
		t.Errorf("no row: %+v %v", res, err)
	}
}
