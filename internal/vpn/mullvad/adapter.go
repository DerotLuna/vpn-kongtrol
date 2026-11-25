// Package mullvad attaches to the provider's official app.
package mullvad

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

func init() {
	appattach.RegisterDesktop(appattach.Desktop{
		Type: "mullvad",
		App: appattach.App{
			DisplayName:  "Mullvad VPN",
			WindowsPaths: appattach.ProgramFolders("Mullvad VPN", "Mullvad VPN.exe"),
			MacApp:       "Mullvad VPN",
			LinuxCmds:    []string{"mullvad-vpn", "/opt/Mullvad VPN/mullvad-vpn"},
		},
		MacExecutable: "Mullvad VPN",
		Matcher:       appattach.Matcher{Patterns: []string{"wg0-mullvad", "wg-mullvad", "Mullvad", "Mullvad Tunnel*"}, Services: []string{"Mullvad VPN*"}, RequireIPv4: true},
		CLI:           appattach.CommandPaths("mullvad", appattach.ProgramFolders("Mullvad VPN", "resources", "mullvad.exe"), []string{"/Applications/Mullvad VPN.app/Contents/Resources/mullvad"}, nil), ConnectArgs: connectArgs,
	})
}
