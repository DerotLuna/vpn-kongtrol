//go:build windows

package vpndetect

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ProtonVPNAppPaths lists the installed Proton VPN app executables, best
// first. The current installer (Inno Setup, github.com/ProtonVPN/win-app
// Setup/SetupBase.iss) puts ProtonVPN.Launcher.exe in
// "%ProgramFiles%\Proton\VPN" and the client in "...\VPN\v<version>\";
// the launcher itself opens the newest client, so it comes first. The
// legacy v2 app lived in "Proton Technologies\ProtonVPN\ProtonVPN.exe".
// Used by the detect probe and by the protonvpn adapter to open the app.
func ProtonVPNAppPaths() []string {
	defaults := []string{filepath.Join(programFiles(), `Proton\VPN`)}
	if lad := os.Getenv("LOCALAPPDATA"); lad != "" {
		// Per-user install folder of Inno Setup's {autopf} (unverified for
		// Proton: its installer requires admin rights by default).
		defaults = append(defaults, filepath.Join(lad, `Programs\Proton\VPN`))
	}
	dirs := installDirs([]string{"Proton VPN", "ProtonVPN"}, defaults...)
	paths := AppPaths(dirs, AppLayout{
		Exes:          []string{"ProtonVPN.Launcher.exe", "ProtonVPN.exe"},
		VersionedExes: []string{"ProtonVPN.Client.exe", "ProtonVPN.exe"},
	})
	return append(paths, AppPaths([]string{filepath.Join(programFilesX86(), `Proton Technologies\ProtonVPN`)},
		AppLayout{Exes: []string{"ProtonVPN.exe"}})...)
}

// NordVPNAppPaths lists the installed NordVPN app executables, best first.
// NordVPN.exe sits in the install folder; its components live in a
// "<version>" subfolder, so a versioned NordVPN.exe is only a fallback.
func NordVPNAppPaths() []string {
	dirs := installDirs([]string{"NordVPN"},
		filepath.Join(programFiles(), "NordVPN"),
		filepath.Join(programFilesX86(), "NordVPN"),
	)
	return AppPaths(dirs, AppLayout{Exes: []string{"NordVPN.exe"}, VersionedExes: []string{"NordVPN.exe"}})
}

// SurfsharkAppPaths lists the installed Surfshark app executables, best
// first (the installer accepts a custom folder, found via the registry).
func SurfsharkAppPaths() []string {
	dirs := installDirs([]string{"Surfshark"},
		filepath.Join(programFiles(), "Surfshark"),
		filepath.Join(programFilesX86(), "Surfshark"),
	)
	return AppPaths(dirs, AppLayout{Exes: []string{"Surfshark.exe"}})
}

// installDirs returns the default folders followed by the install folders
// of "Apps & features" entries whose DisplayName is one of names
// (case-insensitive), without duplicates. Read-only registry access.
func installDirs(names []string, defaults ...string) []string {
	out := slices.Clone(defaults)
	for _, d := range uninstallDirs(names) {
		if !slices.ContainsFunc(out, func(o string) bool { return samePath(o, d) }) {
			out = append(out, d)
		}
	}
	return out
}

// uninstallDirs reads HKLM (64- and 32-bit views) and HKCU
// ...\CurrentVersion\Uninstall for entries named one of names.
func uninstallDirs(names []string) []string {
	const path = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
	roots := []struct {
		key  registry.Key
		view uint32
	}{
		{registry.LOCAL_MACHINE, registry.WOW64_64KEY},
		{registry.LOCAL_MACHINE, registry.WOW64_32KEY},
		{registry.CURRENT_USER, 0},
	}
	var out []string
	for _, r := range roots {
		k, err := registry.OpenKey(r.key, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE|r.view)
		if err != nil {
			continue
		}
		subs, _ := k.ReadSubKeyNames(-1)
		for _, s := range subs {
			sk, err := registry.OpenKey(k, s, registry.QUERY_VALUE|r.view)
			if err != nil {
				continue
			}
			name, _, _ := sk.GetStringValue("DisplayName")
			if slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(strings.TrimSpace(name), n) }) {
				loc, _, _ := sk.GetStringValue("InstallLocation")
				icon, _, _ := sk.GetStringValue("DisplayIcon")
				if d := InstallDirFromEntry(loc, icon); d != "" {
					out = append(out, d)
				}
			}
			sk.Close()
		}
		k.Close()
	}
	return out
}

func programFiles() string    { return envOr("ProgramFiles", `C:\Program Files`) }
func programFilesX86() string { return envOr("ProgramFiles(x86)", `C:\Program Files (x86)`) }

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
