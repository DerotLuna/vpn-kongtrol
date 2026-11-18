package privops

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	dumpPriv = "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk="
	dumpPSK  = "FpCyhws9cxwWoV4xELtfJvjJN+zQVRPISllRWgeopVE="
	dumpPeer = "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
)

// Format from wg(8) "dump": interface line, then one line per peer.
var sampleDump = dumpPriv + "\tHIgo9xNzJMWLKASShiTqIybxZ0U3wGLiUeJ1PKf8ykw=\t51820\toff\n" +
	dumpPeer + "\t" + dumpPSK + "\t203.0.113.7:51820\t10.0.0.0/8\t1700000000\t1234\t5678\t25\n" +
	"TrMvSoP4jYQlY6RIzBgbssQqY3vxI2Pi+y71lOWWXX0=\t(none)\t(none)\t10.9.0.0/24\t0\t0\t148\toff\n"

func TestParseWGDump(t *testing.T) {
	st, err := ParseWGDump(sampleDump)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Peers) != 2 {
		t.Fatalf("peers = %d, want 2", len(st.Peers))
	}
	p := st.Peers[0]
	if p.PublicKey != dumpPeer || p.LatestHandshake != 1700000000 || p.RxBytes != 1234 || p.TxBytes != 5678 || p.PersistentKeepalive != 25 {
		t.Errorf("peer 0 = %+v", p)
	}
	q := st.Peers[1]
	if q.LatestHandshake != 0 || q.TxBytes != 148 || q.PersistentKeepalive != 0 {
		t.Errorf("peer 1 = %+v", q)
	}
}

func TestParseWGDump_CRLFAndNoPeers(t *testing.T) {
	st, err := ParseWGDump(strings.ReplaceAll(strings.SplitN(sampleDump, "\n", 2)[0]+"\n", "\n", "\r\n"))
	if err != nil || len(st.Peers) != 0 || st.Peers == nil {
		t.Fatalf("got %+v, %v; want empty non-nil peers", st, err)
	}
}

func TestParseWGDump_Rejects(t *testing.T) {
	for name, in := range map[string]string{
		"empty":         "",
		"bad iface":     "a\tb\n",
		"short peer":    "a\tb\tc\td\n" + dumpPeer + "\tx\n",
		"bad key":       "a\tb\tc\td\nnot-a-key\t(none)\t(none)\t\t0\t0\t0\toff\n",
		"bad handshake": "a\tb\tc\td\n" + dumpPeer + "\t(none)\t(none)\t\t-1\t0\t0\toff\n",
		"bad rx":        "a\tb\tc\td\n" + dumpPeer + "\t(none)\t(none)\t\t0\tx\t0\toff\n",
		"bad keepalive": "a\tb\tc\td\n" + dumpPeer + "\t(none)\t(none)\t\t0\t0\t0\t99999\n",
	} {
		if _, err := ParseWGDump(in); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
	}
}

func stubDump(t *testing.T, out string, err error) *[]string {
	t.Helper()
	var calls []string
	old := wgDumpFn
	t.Cleanup(func() { wgDumpFn = old })
	wgDumpFn = func(_ context.Context, n string) (string, error) { calls = append(calls, n); return out, err }
	return &calls
}

// The helper only returns peer public keys and counters: never the
// interface private key or a preshared key.
func TestLocalWGStats_ManagedTunnel_NoSecrets(t *testing.T) {
	stubWG(t, func(string) (bool, error) { return true, nil })
	calls := stubDump(t, sampleDump, nil)
	dir := t.TempDir()
	l := NewLocal(LocalOptions{StateDir: dir})
	if err := os.MkdirAll(l.wgDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.wgDir(), "brainbox.conf"), []byte(adoptConf), 0o600); err != nil {
		t.Fatal(err)
	}
	c := pipeClient(l)
	st, err := c.WGStats(context.Background(), WGStatsRequest{Interface: "brainbox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0] != "brainbox" || len(st.Peers) != 2 {
		t.Fatalf("calls %v, stats %+v", *calls, st)
	}
	// Re-encode what crossed the wire and look for secrets.
	b := []byte(strings.Join([]string{st.Peers[0].PublicKey, st.Peers[1].PublicKey}, ","))
	if strings.Contains(string(b), dumpPriv) || strings.Contains(string(b), dumpPSK) {
		t.Fatal("secret key in WGStats reply")
	}
}

func TestLocalWGStats_Validation(t *testing.T) {
	stubWG(t, func(n string) (bool, error) { return n == "rluna", nil })
	calls := stubDump(t, sampleDump, nil)
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	ctx := context.Background()

	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "-bad name"}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("bad name: %v", err)
	}
	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "other"}); denyReason(err) != DenyWGNotManaged {
		t.Errorf("unmanaged without config: %v", err)
	}
	conf := profileConf(t, "rluna")
	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "rluna", ConfigPath: conf}); err != nil {
		t.Errorf("own profile tunnel: %v", err)
	}
	// Read-only: no private copy was adopted.
	if _, err := os.Stat(filepath.Join(l.wgDir(), "rluna.conf")); !os.IsNotExist(err) {
		t.Errorf("WGStats adopted the tunnel: %v", err)
	}
	if len(*calls) != 1 {
		t.Errorf("dump calls = %v, want only the authorized one", *calls)
	}
}

// A dump failure never echoes the command output (it may hold keys).
func TestLocalWGStats_FailureHidesOutput(t *testing.T) {
	stubWG(t, func(string) (bool, error) { return true, nil })
	stubDump(t, "", errors.New("wg show: "+dumpPriv))
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	conf := profileConf(t, "rluna")
	_, err := l.WGStats(context.Background(), WGStatsRequest{Interface: "rluna", ConfigPath: conf})
	if !errors.Is(err, ErrFailed) || strings.Contains(err.Error(), dumpPriv) {
		t.Fatalf("err = %v", err)
	}
}

// A helper from an older Kongtrol answers "unknown operation" (or another
// protocol version): the client reports ErrHelperOutdated.
func TestClientWGStats_OldHelper(t *testing.T) {
	c := NewClientWithDialer(func(ctx context.Context) (net.Conn, error) {
		cl, s := net.Pipe()
		go func() {
			defer s.Close()
			var req Request
			if err := ReadFrame(s, &req); err != nil {
				return
			}
			_ = WriteFrame(s, Response{V: ProtocolVersion, Code: errCodeInvalid, Error: "unknown operation"})
		}()
		return cl, nil
	})
	if _, err := c.WGStats(context.Background(), WGStatsRequest{Interface: "wg0"}); !errors.Is(err, ErrHelperOutdated) {
		t.Fatalf("err = %v, want ErrHelperOutdated", err)
	}
	if !ReadOnlyOp(OpWGStats) || ReadOnlyOp(OpWGInstall) || leasedOp(OpWGStats) {
		t.Error("wg.stats must be read-only and unleased")
	}
}
