//go:build windows

package vpndetect

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FileVersion reads a Windows binary's VS_FIXEDFILEINFO.ProductVersion
// ("6.4.10.1821") with the Win32 version API. Nothing is launched, so GUI
// clients (FortiClient.exe, wireguard.exe) never open. "" when the file
// has no version resource.
func FileVersion(path string) string {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 {
		return ""
	}
	buf := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return ""
	}
	var fixed *windows.VS_FIXEDFILEINFO
	var n uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\`, unsafe.Pointer(&fixed), &n); err != nil || fixed == nil || n == 0 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		fixed.ProductVersionMS>>16, fixed.ProductVersionMS&0xffff,
		fixed.ProductVersionLS>>16, fixed.ProductVersionLS&0xffff)
}
