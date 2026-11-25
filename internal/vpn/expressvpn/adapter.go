// Package expressvpn attaches to the provider's official app.
package expressvpn

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

func init() {
	appattach.RegisterDesktop(appattach.Desktop{
		Type: "expressvpn",
		App: appattach.App{
			DisplayName:  "ExpressVPN",
			WindowsPaths: appattach.ProgramFolders("ExpressVPN", "ExpressVPN.exe"),
			MacApp:       "ExpressVPN",
			LinuxCmds:    []string{"expressvpn-app", "expressvpn"},
		},
		MacExecutable: "ExpressVPN",
		Matcher:       appattach.Matcher{Patterns: []string{"ExpressVPN Tap Adapter*", "ExpressVPN TUN*", "ExpressVPN WinTUN*", "expressvpn*"}, Services: []string{"ExpressVPN*"}, RequireIPv4: true},
		CLI:           cli(), ConnectArgs: connectArgs,
	})
}
