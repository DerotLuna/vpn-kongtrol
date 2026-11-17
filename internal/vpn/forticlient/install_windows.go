//go:build windows

package forticlient

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// DetectInstall finds FortiClient without launching anything: the
// app_path override's folder, else the installer's INSTALLDIR registry
// value (HKLM\SOFTWARE\Fortinet\FortiClient, seen on 6.4.10), else the
// default folder. The version is FortiClient.exe's ProductVersion
// resource, read with the Win32 version API (no PowerShell).
func DetectInstall(appPath string) Install {
	dir := ""
	if appPath != "" {
		dir = filepath.Dir(appPath)
	} else if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Fortinet\FortiClient`, registry.QUERY_VALUE); err == nil {
		if v, _, err := k.GetStringValue("INSTALLDIR"); err == nil && v != "" {
			dir = filepath.Clean(v)
		}
		k.Close()
	}
	if dir == "" {
		dir = filepath.Dir(defaultWindowsBinary)
	}
	in := Install{Path: filepath.Join(dir, "FortiClient.exe")}
	if _, err := os.Stat(in.Path); err != nil {
		return Install{}
	}
	in.Version = productVersion(in.Path)
	if fv := filepath.Join(dir, "FortiVPN.exe"); fileExists(fv) {
		in.FortiVPN = fv
	}
	return in
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// productVersion reads VS_FIXEDFILEINFO.ProductVersion ("6.4.10.1821").
func productVersion(path string) string {
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
