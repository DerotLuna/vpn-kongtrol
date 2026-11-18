//go:build windows

package forticlient

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"

	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
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
	in.Version = vpndetect.FileVersion(in.Path)
	if fv := filepath.Join(dir, "FortiVPN.exe"); fileExists(fv) {
		in.FortiVPN = fv
	}
	return in
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
