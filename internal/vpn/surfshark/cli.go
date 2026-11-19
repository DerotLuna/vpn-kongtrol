package surfshark

import (
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// Surfshark has no command-line interface on Windows or macOS, and the
// current Linux app is GUI-only (the legacy Linux `surfshark-vpn` tool is
// interactive and superseded), so there is no automatic connect /
// disconnect path: appattach.Provider.AutoConnect and Disconnect stay nil.

// app is the official Surfshark app per OS. Install locations are best
// effort; `app_path:` overrides them. On Windows the same lookup as the
// detect list (Program Files, Program Files (x86), registry install folder).
var app = appattach.App{
	DisplayName: "Surfshark",
	WindowsFind: vpndetect.SurfsharkAppPaths,
	MacApp:      "Surfshark",
	LinuxCmds:   []string{"surfshark"},
}
