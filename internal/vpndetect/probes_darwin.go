//go:build darwin

package vpndetect

import "path/filepath"

// standards mirrors the paths each adapter package executes on macOS.
var standards = map[string]standard{
	"forticlient":     {locations: []string{"/Applications/FortiClient.app/Contents/MacOS/FortiClientAgent"}, versionArgs: []string{"--version"}},
	"ciscoanyconnect": {locations: []string{"/opt/cisco/anyconnect/bin/vpn"}, versionArgs: []string{"-v"}},
	"globalprotect":   {locations: []string{"/Applications/GlobalProtect.app/Contents/MacOS/globalprotect"}},
	"wireguard":       {locations: []string{"wg-quick"}},
	"tailscale":       {locations: []string{"tailscale"}, versionArgs: []string{"version"}},
	"cloudflarewarp":  {locations: []string{"warp-cli", "/Applications/Cloudflare WARP.app/Contents/MacOS/warp-cli"}, versionArgs: []string{"--version"}},
	// protonvpn: protonvpn-cli is Linux-only; the adapter refuses to connect here.
	// nordvpn: the macOS app has no CLI; the adapter refuses to connect here.
}

var probes = []Probe{
	{
		Label: "FortiClient", AdapterType: "forticlient",
		PathNames:   []string{"fortivpn", "FortiClient"},
		SearchDirs:  []string{"/Applications/FortiClient.app/Contents/MacOS"},
		SearchDepth: 2,
		ExeNames:    []string{"FortiClient", "FortiClientAgent"},
	},
	{
		Label: "OpenVPN", AdapterType: "openvpn",
		PathNames:   []string{"openvpn"},
		SearchDirs:  []string{"/usr/local/sbin", "/opt/homebrew/sbin", "/Applications/Tunnelblick.app/Contents/Resources"},
		SearchDepth: 2,
		ExeNames:    []string{"openvpn"},
		VersionArgs: []string{"--version"},
	},
	{
		Label: "ProtonVPN", AdapterType: "protonvpn",
		PathNames:   []string{"protonvpn-cli", "protonvpn"},
		SearchDirs:  []string{"/Applications/ProtonVPN.app/Contents/MacOS"},
		SearchDepth: 1,
		ExeNames:    []string{"ProtonVPN"},
	},
	{
		Label: "NordVPN", AdapterType: "nordvpn",
		SearchDirs:  []string{"/Applications/NordVPN.app/Contents/MacOS"},
		SearchDepth: 1,
		ExeNames:    []string{"NordVPN"},
	},
	{
		Label: "Cisco AnyConnect", AdapterType: "ciscoanyconnect",
		PathNames:   []string{"vpn"},
		SearchDirs:  []string{"/opt/cisco"},
		SearchDepth: 3,
		ExeNames:    []string{"vpn"},
		VersionArgs: []string{"-v"},
	},
	{
		Label: "WireGuard", AdapterType: "wireguard",
		PathNames:   []string{"wg-quick", "wg"},
		SearchDirs:  []string{"/usr/local/bin", "/opt/homebrew/bin", "/Applications/WireGuard.app/Contents/MacOS"},
		SearchDepth: 1,
		ExeNames:    []string{"wg-quick", "wg", "WireGuard"},
	},
	{
		Label: "GlobalProtect", AdapterType: "globalprotect",
		PathNames:   []string{"globalprotect"},
		SearchDirs:  []string{"/Applications/GlobalProtect.app/Contents/MacOS"},
		SearchDepth: 2,
		ExeNames:    []string{"GlobalProtect", "globalprotect"},
	},
	{
		Label: "Tailscale", AdapterType: "tailscale",
		PathNames:   []string{"tailscale"},
		SearchDirs:  []string{"/Applications/Tailscale.app/Contents/MacOS"},
		SearchDepth: 1,
		ExeNames:    []string{"tailscale", "Tailscale"},
		VersionArgs: []string{"version"},
	},
	{
		Label: "Cloudflare WARP", AdapterType: "cloudflarewarp",
		PathNames:   []string{"warp-cli"},
		SearchDirs:  []string{"/Applications/Cloudflare WARP.app/Contents/MacOS"},
		SearchDepth: 1,
		ExeNames:    []string{"warp-cli"},
		VersionArgs: []string{"--version"},
	},
	{
		Label:       "TunnelBear",
		PathNames:   []string{"tunnelbear"},
		SearchDirs:  []string{"/Applications/TunnelBear.app/Contents/MacOS"},
		SearchDepth: 2,
		ExeNames:    []string{"TunnelBear"},
	},
}

func samePath(a, b string) bool { return filepath.Clean(a) == filepath.Clean(b) }
