package privops

import (
	"context"
	"net/netip"
	"testing"
)

// Key pair from the wg(8) / wg-quick(8) man page example.
func TestWGConfigPublicKey_ManPageExample(t *testing.T) {
	got, ok := wgConfigPublicKey([]byte(adoptConf))
	if !ok || got != "HIgo9xNzJMWLKASShiTqIybxZ0U3wGLiUeJ1PKf8ykw=" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := wgConfigPublicKey([]byte("[Peer]\nPrivateKey = " + dumpPriv + "\n")); ok {
		t.Fatal("PrivateKey outside [Interface] must not count")
	}
	if _, ok := wgConfigPublicKey([]byte("[Interface]\nPrivateKey = nope\n")); ok {
		t.Fatal("bad key accepted")
	}
}

// Regression: a caller owning rluna.conf could read the counters of, change
// the AllowedIPs of, or remove a foreign tunnel that happened to be named
// "rluna". The running interface must use the config's key pair.
func TestWGAuthorize_ForeignSameNameTunnelDenied(t *testing.T) {
	stubWG(t, func(n string) (bool, error) { return n == "rluna", nil })
	foreign := "(hidden)\tTrMvSoP4jYQlY6RIzBgbssQqY3vxI2Pi+y71lOWWXX0=\t51820\toff\n"
	stubDump(t, foreign, nil)
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	ctx := context.Background()
	conf := profileConf(t, "rluna")

	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "rluna", ConfigPath: conf}); denyReason(err) != DenyWGNotManaged {
		t.Errorf("stats: %v", err)
	}
	if err := l.WGRemoveTunnel(ctx, WGRemoveRequest{Name: "rluna", ConfigPath: conf}); denyReason(err) != DenyWGNotManaged {
		t.Errorf("remove: %v", err)
	}
	req := WGAllowedIPsRequest{Interface: "rluna", ConfigPath: conf, PeerPublicKey: dumpPeer, AllowedIPs: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	if err := l.WGSetAllowedIPs(ctx, req); denyReason(err) != DenyWGNotManaged {
		t.Errorf("allowed ips: %v", err)
	}
	if l.wgManaged("rluna") {
		t.Fatal("foreign tunnel adopted")
	}
}

func TestValidateWGName_Reserved(t *testing.T) {
	for _, n := range []string{"all", "ALL", "interfaces", "Interfaces"} {
		if err := ValidateWGName(n); err == nil {
			t.Errorf("%q accepted", n)
		}
	}
	if err := ValidateWGName("all2"); err != nil {
		t.Errorf("all2: %v", err)
	}
}

// Success of wg.stats authorized by a profile config is audited, at most
// once per interface per hour; a managed tunnel's poll is not.
func TestWGStats_AuditMark(t *testing.T) {
	stubWG(t, func(string) (bool, error) { return true, nil })
	stubDump(t, sampleDump, nil)
	l := NewLocal(LocalOptions{StateDir: t.TempDir()})
	conf := profileConf(t, "rluna")

	ctx, mark := WithAuditMark(context.Background())
	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "rluna", ConfigPath: conf}); err != nil {
		t.Fatal(err)
	}
	if !mark.Set() {
		t.Fatal("first config-authorized stats not marked for audit")
	}
	ctx, mark = WithAuditMark(context.Background())
	if _, err := l.WGStats(ctx, WGStatsRequest{Interface: "rluna", ConfigPath: conf}); err != nil {
		t.Fatal(err)
	}
	if mark.Set() {
		t.Fatal("audit not throttled")
	}
}
