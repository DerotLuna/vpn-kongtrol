package privops

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestCheckDefaultEgressReadsConfigWithoutChangingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wg0.conf")
	if err := os.WriteFile(path, []byte(dualStackConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := CheckDefaultEgress("wireguard", path, ""); !got.Compatible {
		t.Fatalf("check = %+v", got)
	}
	if got := CheckDefaultEgress("openvpn", path, ""); got.Compatible || got.Reason != "unsupported_profile" {
		t.Fatalf("OpenVPN check = %+v", got)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != dualStackConfig {
		t.Fatalf("profile file changed: %v", err)
	}
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	if got := CheckDefaultEgress("wireguard", path, ""); got.Compatible || got.Reason != "config_unavailable" {
		t.Fatalf("writable file check = %+v", got)
	}
}

func TestConflictingNativeDefaultRules(t *testing.T) {
	const mark = 51820
	main := netlink.Rule{Table: unix.RT_TABLE_MAIN, SuppressPrefixlen: -1}
	suppressed := netlink.Rule{Table: unix.RT_TABLE_MAIN, SuppressPrefixlen: 0}
	own := netlink.Rule{Table: mark, Mark: mark, Invert: true}
	for _, test := range []struct {
		name      string
		rules     []netlink.Rule
		replacing int
		want      bool
	}{
		{"standard main table", []netlink.Rule{main}, 0, false},
		{"existing automatic default", []netlink.Rule{main, suppressed, own}, 0, true},
		{"replacing managed default", []netlink.Rule{main, suppressed, own}, mark, false},
		{"foreign automatic default", []netlink.Rule{main, suppressed, own}, mark + 1, true},
		{"duplicate suppression", []netlink.Rule{main, suppressed, suppressed, own}, mark, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := conflictingNativeDefaultRules(test.rules, test.replacing); got != test.want {
				t.Fatalf("conflict = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDefaultEgressMarkerProtectsAllowedIPs(t *testing.T) {
	stub := stubWG(t, func(string) (bool, error) { return true, nil })
	local := NewLocal(LocalOptions{StateDir: t.TempDir()})
	if err := ensurePrivateDir(local.wgDir()); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateFile(filepath.Join(local.wgDir(), "wg0.conf"), []byte(dualStackConfig)); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateFile(local.wgDefaultEgressMarker("wg0"), nil); err != nil {
		t.Fatal(err)
	}
	req := WGAllowedIPsRequest{
		Interface:     "wg0",
		PeerPublicKey: "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=",
		AllowedIPs:    []netip.Prefix{netip.MustParsePrefix("10.7.0.0/24")},
	}
	if err := local.WGSetAllowedIPs(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(stub.sets) != 0 {
		t.Fatalf("helper modified native peer routes: %v", stub.sets)
	}
	if err := local.WGRemoveTunnel(context.Background(), WGRemoveRequest{Name: "wg0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(local.wgDefaultEgressMarker("wg0")); !os.IsNotExist(err) {
		t.Fatalf("marker survived after tunnel removal: %v", err)
	}
}
