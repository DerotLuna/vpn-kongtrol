// Package cyberghost attaches to the provider's official app.
package cyberghost

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

func init() {
	appattach.RegisterDesktop(appattach.Desktop{
		Type: "cyberghost",
		App: appattach.App{
			DisplayName:  "CyberGhost VPN",
			WindowsPaths: appattach.ProgramFolders("CyberGhost 8", "Dashboard.exe"),
			MacApp:       "CyberGhost VPN",
			LinuxCmds:    []string{},
		},
		MacExecutable: "CyberGhost VPN",
		Matcher:       appattach.Matcher{Patterns: []string{"CyberGhost WireGuard*", "CyberGhost VPN*", "CyberGhost TAP*", "cyberghost*"}, Services: []string{"CyberGhost*"}, RequireIPv4: true},
		DetectPaths:   detectPaths(),
	})
}
