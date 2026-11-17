package surfshark

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

// Surfshark has no command-line interface on Windows or macOS, and the
// current Linux app is GUI-only (the legacy Linux `surfshark-vpn` tool is
// interactive and superseded), so there is no automatic connect /
// disconnect path: appattach.Provider.AutoConnect and Disconnect stay nil.

// app is the official Surfshark app per OS. Install locations are best
// effort; `app_path:` overrides them.
var app = appattach.App{
	DisplayName: "Surfshark",
	WindowsPaths: []string{
		`C:\Program Files\Surfshark\Surfshark.exe`,
		`C:\Program Files (x86)\Surfshark\Surfshark.exe`,
	},
	MacApp:    "Surfshark",
	LinuxCmds: []string{"surfshark"},
}
