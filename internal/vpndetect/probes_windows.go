//go:build windows

package vpndetect

import (
	"path/filepath"
	"strings"
)

// standards mirrors the paths each adapter package executes on Windows.
var standards = map[string]standard{
	"forticlient":     {locations: []string{`C:\Program Files\Fortinet\FortiClient\FortiClient.exe`}},
	"ciscoanyconnect": {locations: []string{`C:\Program Files\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`}, versionArgs: []string{"-v"}},
	"globalprotect":   {locations: []string{`C:\Program Files\Palo Alto Networks\GlobalProtect\globalprotect.exe`}},
	"wireguard":       {locations: []string{`C:\Program Files\WireGuard\wireguard.exe`}},
	"tailscale":       {locations: []string{`C:\Program Files\Tailscale\tailscale.exe`}, versionArgs: []string{"version"}},
	"cloudflarewarp":  {locations: []string{"warp-cli", `C:\Program Files\Cloudflare\Cloudflare WARP\warp-cli.exe`}, versionArgs: []string{"--version"}},
	"nordvpn":         {locations: []string{`C:\Program Files\NordVPN\NordVPN.exe`}},
	// protonvpn: protonvpn-cli is Linux-only; the adapter refuses to connect here.
}

var probes = []Probe{
	{
		Label: "FortiClient", AdapterType: "forticlient",
		PathNames:   []string{"FortiClient"},
		SearchDirs:  []string{`C:\Program Files\Fortinet`, `C:\Program Files (x86)\Fortinet`},
		SearchDepth: 2,
		ExeNames:    []string{"FortiClient.exe"},
	},
	{
		Label: "OpenVPN", AdapterType: "openvpn",
		PathNames: []string{"openvpn"},
		SearchDirs: []string{
			`C:\Program Files\OpenVPN`, `C:\Program Files (x86)\OpenVPN`,
			`C:\Program Files\OpenVPN Connect`, `C:\Program Files (x86)\OpenVPN Connect`,
		},
		SearchDepth: 2,
		ExeNames:    []string{"openvpn.exe", "OpenVPNConnect.exe"},
		VersionArgs: []string{"--version"},
	},
	{
		Label: "ProtonVPN", AdapterType: "protonvpn",
		PathNames:   []string{"protonvpn-cli", "protonvpn"},
		SearchDirs:  []string{`C:\Program Files\Proton`, `C:\Program Files (x86)\Proton`, `C:\Program Files (x86)\Proton Technologies`},
		SearchDepth: 3,
		ExeNames:    []string{"ProtonVPN.Launcher.exe", "ProtonVPN.Client.exe", "ProtonVPN.exe"},
	},
	{
		Label: "NordVPN", AdapterType: "nordvpn",
		SearchDirs:  []string{`C:\Program Files\NordVPN`, `C:\Program Files (x86)\NordVPN`},
		SearchDepth: 1,
		ExeNames:    []string{"NordVPN.exe"},
	},
	{
		Label: "Cisco AnyConnect", AdapterType: "ciscoanyconnect",
		PathNames:   []string{"vpncli"},
		SearchDirs:  []string{`C:\Program Files\Cisco`, `C:\Program Files (x86)\Cisco`},
		SearchDepth: 3,
		ExeNames:    []string{"vpncli.exe"},
		VersionArgs: []string{"-v"},
	},
	{
		Label: "WireGuard", AdapterType: "wireguard",
		PathNames:   []string{"wireguard"},
		SearchDirs:  []string{`C:\Program Files\WireGuard`},
		SearchDepth: 1,
		ExeNames:    []string{"wireguard.exe"},
	},
	{
		Label: "GlobalProtect", AdapterType: "globalprotect",
		SearchDirs:  []string{`C:\Program Files\Palo Alto Networks`},
		SearchDepth: 2,
		ExeNames:    []string{"PanGPA.exe", "pangpcrypt.exe"},
	},
	{
		Label: "Tailscale", AdapterType: "tailscale",
		PathNames:   []string{"tailscale"},
		SearchDirs:  []string{`C:\Program Files\Tailscale`},
		SearchDepth: 1,
		ExeNames:    []string{"tailscale.exe"},
		VersionArgs: []string{"version"},
	},
	{
		Label: "Cloudflare WARP", AdapterType: "cloudflarewarp",
		PathNames:   []string{"warp-cli"},
		SearchDirs:  []string{`C:\Program Files\Cloudflare\Cloudflare WARP`, `C:\Program Files (x86)\Cloudflare\Cloudflare WARP`},
		SearchDepth: 1,
		ExeNames:    []string{"warp-cli.exe"},
		VersionArgs: []string{"--version"},
	},
	{
		Label:       "TunnelBear",
		SearchDirs:  []string{`C:\Program Files\TunnelBear`, `C:\Program Files (x86)\TunnelBear`},
		SearchDepth: 2,
		ExeNames:    []string{"TunnelBear.exe"},
	},
}

// samePath compares paths the way NTFS does: case-insensitively.
func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
