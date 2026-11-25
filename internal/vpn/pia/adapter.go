// Package pia attaches to the provider's official app.
package pia

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

func init() {
	appattach.RegisterDesktop(appattach.Desktop{
		Type: "pia",
		App: appattach.App{
			DisplayName:  "Private Internet Access",
			WindowsPaths: appattach.ProgramFolders("Private Internet Access", "pia-client.exe"),
			MacApp:       "Private Internet Access",
			LinuxCmds:    []string{"pia-client", "/opt/piavpn/bin/pia-client"},
		},
		MacExecutable: "pia-client",
		Matcher:       appattach.Matcher{Patterns: []string{"wgpia0", "PIA OpenVPN WinTUN Adapter*", "Private Internet Access Network Adapter*"}, Services: []string{"Private Internet Access*"}, RequireIPv4: true},
		CLI:           appattach.CommandPaths("piactl", appattach.ProgramFolders("Private Internet Access", "piactl.exe"), []string{"/Applications/Private Internet Access.app/Contents/MacOS/piactl"}, []string{"/opt/piavpn/bin/piactl"}), ConnectArgs: connectArgs,
	})
}
