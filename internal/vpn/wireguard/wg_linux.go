//go:build linux

package wireguard

import (
	"net"

	"github.com/vishvananda/netlink"
)

// wgPreUp attaches to an existing kernel WireGuard link without invoking
// wg-quick again. An ordinary interface with the same name never qualifies.
func wgPreUp(name string) bool {
	link, err := netlink.LinkByName(name)
	return err == nil && kernelWireGuardLinkUp(link)
}

func kernelWireGuardLinkUp(link netlink.Link) bool {
	return link != nil && link.Type() == "wireguard" && link.Attrs().Flags&net.FlagUp != 0
}

func tunnelServiceRunning(string) bool { return false }
