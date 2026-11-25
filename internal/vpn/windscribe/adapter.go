// Package windscribe attaches to the provider's official app.
package windscribe

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

func init() {
	appattach.RegisterDesktop(appattach.Desktop{
		Type: "windscribe",
		App: appattach.App{
			DisplayName:  "Windscribe",
			WindowsPaths: appattach.ProgramFolders("Windscribe", "Windscribe.exe"),
			MacApp:       "Windscribe",
			LinuxCmds:    []string{"windscribe"},
		},
		MacExecutable: "Windscribe",
		Matcher:       appattach.Matcher{Patterns: []string{"WindscribeVPN", "WindscribeWireguard", "Windscribe WireGuard*", "Windscribe TAP*", "wswg*"}, Services: []string{"Windscribe*"}, RequireIPv4: true},
		CLI:           appattach.CommandPaths("windscribe-cli", appattach.ProgramFolders("Windscribe", "windscribe-cli.exe"), []string{"/Applications/Windscribe.app/Contents/MacOS/windscribe-cli"}, nil), ConnectArgs: connectArgs,
	})
}
