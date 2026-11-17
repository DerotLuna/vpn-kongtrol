package surfshark

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

// matcher finds the Surfshark tunnel. Windows: Surfshark's support docs
// name a "Surfshark DCO adapter"; WireGuard / TAP adapters also carry
// "Surfshark" in their name or description (best guess). Linux: interface
// names starting with "surfshark" (best guess, e.g. surfshark_wg). macOS:
// the app's Network Extension service runs on a generic utunN, so it is
// found through `scutil --nc list` by service name. `interface:` overrides
// all of this.
var matcher = appattach.Matcher{
	Patterns: []string{"*surfshark*"},
	Services: []string{"surfshark*"},
}
