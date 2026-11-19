//go:build linux

package sockowner

import (
	"os"
	"path/filepath"
	"testing"
)

const procTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2605 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1002        0 11111 1 0000000000000000 100 0 0 10 0
`

func TestCheckLinuxOtherUser(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tcp")
	if err := os.WriteFile(p, []byte(procTCP), 0o600); err != nil {
		t.Fatal(err)
	}
	oldF, oldU := procFiles, getuid
	t.Cleanup(func() { procFiles, getuid = oldF, oldU })
	procFiles = []string{p}
	client, server := ap("127.0.0.1:50000"), ap("127.0.0.1:9733") // 0x2605
	for uid, want := range map[int]Verdict{1001: Other, 1002: Self} {
		getuid = func() int { return uid }
		if res, err := Check(client, server); err != nil || res.Verdict != want {
			t.Errorf("uid %d: %+v %v, want %v", uid, res, err, want)
		}
	}
}
