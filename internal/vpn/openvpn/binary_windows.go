//go:build windows

package openvpn

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// platformCandidates returns the locations the official OpenVPN installer
// uses on Windows. The installer records the exact path in
// HKLM\SOFTWARE\OpenVPN\exe_path, which also covers custom install dirs.
func platformCandidates() []string {
	var out []string
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\OpenVPN`, registry.QUERY_VALUE); err == nil {
		if p, _, err := k.GetStringValue("exe_path"); err == nil {
			out = append(out, p)
		}
		k.Close()
	}
	for _, env := range []string{"ProgramFiles", "ProgramW6432", "ProgramFiles(x86)"} {
		if dir := os.Getenv(env); dir != "" {
			out = append(out, filepath.Join(dir, "OpenVPN", "bin", "openvpn.exe"))
		}
	}
	return out
}
