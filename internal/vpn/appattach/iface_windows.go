//go:build windows

package appattach

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// osInterfaces lists adapters with friendly name, description and
// operational status via GetAdaptersAddresses: provider adapters are often
// recognisable only by description (a TAP adapter named "Ethernet 3" is
// described as "TAP-NordVPN Windows Adapter V9"). Falls back to
// net.Interfaces when the call fails.
func osInterfaces() []Iface {
	size := uint32(15 * 1024)
	for range 3 {
		buf := make([]byte, size)
		aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_PREFIX, 0, aa, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			break
		}
		var out []Iface
		for a := aa; a != nil; a = a.Next {
			ifc := Iface{
				Name:        windows.UTF16PtrToString(a.FriendlyName),
				Description: windows.UTF16PtrToString(a.Description),
				Up:          a.OperStatus == windows.IfOperStatusUp,
			}
			for u := a.FirstUnicastAddress; u != nil; u = u.Next {
				if ip := u.Address.IP(); ip != nil {
					ifc.Addrs = append(ifc.Addrs, ip)
				}
			}
			out = append(out, ifc)
		}
		return out
	}
	return netInterfaces()
}

// osServiceTunnel: macOS-only probe; nothing on Windows.
func osServiceTunnel([]string) (Tunnel, bool) { return Tunnel{}, false }
