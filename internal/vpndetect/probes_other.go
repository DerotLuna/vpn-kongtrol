//go:build !windows && !darwin

package vpndetect

import "path/filepath"

// standards mirrors the paths each adapter package executes on Linux.
var standards = map[string]standard{
	"forticlient":     {locations: []string{"forticlientsslvpn"}, versionArgs: []string{"--version"}},
	"ciscoanyconnect": {locations: []string{"/opt/cisco/anyconnect/bin/vpn"}, versionArgs: []string{"-v"}},
	"wireguard":       {locations: []string{"wg-quick"}},
	"tailscale":       {locations: []string{"tailscale"}, versionArgs: []string{"version"}},
	"cloudflarewarp":  {locations: []string{"warp-cli", "/usr/bin/warp-cli", "/usr/local/bin/warp-cli"}, versionArgs: []string{"--version"}},
	"protonvpn":       {locations: []string{"protonvpn", "protonvpn-cli"}, versionArgs: []string{"--version"}},
	"nordvpn":         {locations: []string{"nordvpn"}, versionArgs: []string{"--version"}},
	// globalprotect: the adapter does not support Linux.
}

var probes = []Probe{
	{
		Label: "FortiClient", AdapterType: "forticlient",
		PathNames:   []string{"fortivpn", "forticlientsslvpn", "FortiClient"},
		SearchDirs:  []string{"/opt/forticlient", "/opt/fortinet/forticlient", "/usr/share/forticlient", "/usr/lib/forticlient"},
		SearchDepth: 2,
		ExeNames:    []string{"forticlientsslvpn", "FortiClient", "forticlient"},
	},
	{
		Label: "NordVPN", AdapterType: "nordvpn",
		PathNames:   []string{"nordvpn"},
		VersionArgs: []string{"--version"},
	},
	{
		Label: "OpenVPN", AdapterType: "openvpn",
		PathNames:   []string{"openvpn"},
		SearchDirs:  []string{"/usr/sbin", "/usr/local/sbin"},
		SearchDepth: 0,
		ExeNames:    []string{"openvpn"},
		VersionArgs: []string{"--version"},
	},
	{
		Label: "ProtonVPN", AdapterType: "protonvpn",
		PathNames:   []string{"protonvpn", "protonvpn-cli"},
		SearchDirs:  []string{"/opt/protonvpn"},
		SearchDepth: 2,
		ExeNames:    []string{"protonvpn-cli", "protonvpn"},
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
		PathNames: []string{"wg-quick", "wg"},
	},
	{
		Label: "GlobalProtect", AdapterType: "globalprotect",
		PathNames:   []string{"globalprotect"},
		SearchDirs:  []string{"/opt/paloaltonetworks"},
		SearchDepth: 2,
		ExeNames:    []string{"globalprotect", "pangpcrypt"},
	},
	{
		Label: "Tailscale", AdapterType: "tailscale",
		PathNames:   []string{"tailscale"},
		SearchDirs:  []string{"/snap/tailscale/current/bin"},
		SearchDepth: 0,
		ExeNames:    []string{"tailscale"},
		VersionArgs: []string{"version"},
	},
	{
		Label: "Cloudflare WARP", AdapterType: "cloudflarewarp",
		PathNames:   []string{"warp-cli"},
		VersionArgs: []string{"--version"},
	},
	{
		Label:     "TunnelBear",
		PathNames: []string{"tunnelbear"},
	},
}

func samePath(a, b string) bool { return filepath.Clean(a) == filepath.Clean(b) }
