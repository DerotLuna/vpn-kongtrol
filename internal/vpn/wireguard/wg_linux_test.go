//go:build linux

package wireguard

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestKernelWireGuardLinkUp(t *testing.T) {
	attrs := netlink.LinkAttrs{Flags: net.FlagUp}
	if !kernelWireGuardLinkUp(&netlink.GenericLink{LinkAttrs: attrs, LinkType: "wireguard"}) {
		t.Fatal("existing WireGuard interface should be reusable")
	}
	if kernelWireGuardLinkUp(&netlink.GenericLink{LinkAttrs: attrs, LinkType: "dummy"}) {
		t.Fatal("ordinary interface should not be mistaken for WireGuard")
	}
	if kernelWireGuardLinkUp(&netlink.GenericLink{LinkType: "wireguard"}) {
		t.Fatal("down WireGuard interface should not be reused")
	}
}
