package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ── VPN client detection ──────────────────────────────────────────────────────

type detectedVPN struct {
	label   string
	version string
}

// vpnProbe describes how to locate one VPN client.
type vpnProbe struct {
	label       string
	adapterKey  string
	binaries    []string
	searchDirs  []string
	searchDepth int
	exeNames    []string
	args        []string
}

var vpnProbes = []vpnProbe{
	{
		label:      "FortiClient",
		adapterKey: "forticlient",
		binaries:   []string{"fortivpn", "forticlientsslvpn", "FortiClient"},
		searchDirs: []string{
			`C:\Program Files\Fortinet`,
			`C:\Program Files (x86)\Fortinet`,
			"/opt/forticlient",
			"/opt/fortinet/forticlient",
			"/usr/share/forticlient",
			"/usr/lib/forticlient",
			"/Applications/FortiClient.app/Contents/MacOS",
		},
		searchDepth: 2,
		exeNames:    []string{"FortiClient.exe", "FortiClient", "forticlient"},
		args:        []string{},
	},
	{
		label:      "OpenVPN",
		adapterKey: "openvpn",
		binaries:   []string{"openvpn"},
		searchDirs: []string{
			`C:\Program Files\OpenVPN`,
			`C:\Program Files (x86)\OpenVPN`,
			`C:\Program Files\OpenVPN Connect`,
			`C:\Program Files (x86)\OpenVPN Connect`,
			"/usr/sbin",
			"/usr/local/sbin",
			"/opt/homebrew/sbin",
			"/Applications/Tunnelblick.app/Contents/Resources",
		},
		searchDepth: 2,
		exeNames:    []string{"openvpn.exe", "OpenVPNConnect.exe", "openvpn"},
		args:        []string{"--version"},
	},
	{
		label:      "ProtonVPN",
		adapterKey: "protonvpn",
		binaries:   []string{"protonvpn-cli", "protonvpn"},
		searchDirs: []string{
			`C:\Program Files\Proton`,
			`C:\Program Files (x86)\Proton`,
			`C:\Program Files (x86)\Proton Technologies`,
			"/usr/bin",
			"/usr/local/bin",
			"/opt/protonvpn",
			"/Applications/ProtonVPN.app/Contents/MacOS",
		},
		searchDepth: 3,
		exeNames:    []string{"ProtonVPN.Launcher.exe", "ProtonVPN.Client.exe", "ProtonVPN.exe", "protonvpn-cli", "protonvpn"},
		args:        []string{},
	},
	{
		label:      "Cisco AnyConnect",
		adapterKey: "ciscoanyconnect",
		binaries:   []string{"vpn", "vpncli"},
		searchDirs: []string{
			"/opt/cisco",
			`C:\Program Files\Cisco`,
			`C:\Program Files (x86)\Cisco`,
		},
		searchDepth: 3,
		exeNames:    []string{"vpncli.exe", "vpncli", "vpn"},
		args:        []string{"-v"},
	},
	{
		label:      "WireGuard",
		adapterKey: "wireguard",
		binaries:   []string{"wg", "wg-quick", "wireguard"},
		searchDirs: []string{
			`C:\Program Files\WireGuard`,
			"/usr/local/bin",
			"/opt/homebrew/bin",
			"/Applications/WireGuard.app/Contents/MacOS",
		},
		searchDepth: 1,
		exeNames:    []string{"wireguard.exe", "wg.exe", "wg", "wireguard"},
		args:        []string{"--version"},
	},
	{
		label:      "GlobalProtect",
		adapterKey: "globalprotect",
		binaries:   []string{"globalprotect", "pangpcrypt"},
		searchDirs: []string{
			"/opt/paloaltonetworks",
			`C:\Program Files\Palo Alto Networks`,
			"/Applications/GlobalProtect.app/Contents/MacOS",
		},
		searchDepth: 2,
		exeNames:    []string{"PanGPA.exe", "pangpcrypt.exe", "GlobalProtect", "globalprotect", "pangpcrypt"},
		args:        []string{"--version"},
	},
	{
		label:      "Tailscale",
		adapterKey: "tailscale",
		binaries:   []string{"tailscale"},
		searchDirs: []string{
			`C:\Program Files\Tailscale`,
			"/Applications/Tailscale.app/Contents/MacOS",
			"/snap/tailscale/current/bin",
		},
		searchDepth: 1,
		exeNames:    []string{"tailscale.exe", "tailscale"},
		args:        []string{"version"},
	},
	{
		label:      "Cloudflare WARP",
		adapterKey: "cloudflarewarp",
		binaries:   []string{"warp-cli"},
		searchDirs: []string{
			`C:\Program Files\Cloudflare\Cloudflare WARP`,
			`C:\Program Files (x86)\Cloudflare\Cloudflare WARP`,
			"/Applications/Cloudflare WARP.app/Contents/MacOS",
			"/usr/bin",
			"/usr/local/bin",
		},
		searchDepth: 1,
		exeNames:    []string{"warp-cli.exe", "warp-cli"},
		args:        []string{"--version"},
	},
	{
		label:      "TunnelBear",
		adapterKey: "",
		binaries:   []string{"tunnelbear"},
		searchDirs: []string{
			`C:\Program Files\TunnelBear`,
			`C:\Program Files (x86)\TunnelBear`,
			"/Applications/TunnelBear.app/Contents/MacOS",
		},
		searchDepth: 2,
		exeNames:    []string{"TunnelBear.exe", "TunnelBear"},
		args:        []string{"--version"},
	},
}

func detectInstalledVPNs() []detectedVPN {
	var found []detectedVPN
	for _, p := range vpnProbes {
		if d, ok := resolveProbe(p); ok {
			found = append(found, d)
		}
	}
	return found
}

func resolveProbe(p vpnProbe) (detectedVPN, bool) {
	for _, bin := range p.binaries {
		path, err := exec.LookPath(bin)
		if err != nil {
			if _, statErr := os.Stat(bin); statErr != nil {
				continue
			}
			path = bin
		}
		return detectedVPN{label: p.label, version: versionOf(path, p.args)}, true
	}

	var candidates []string
	for _, dir := range p.searchDirs {
		walkExe(dir, p.exeNames, p.searchDepth, &candidates)
	}
	if len(candidates) == 0 {
		return detectedVPN{}, false
	}
	sort.Strings(candidates)
	path := candidates[len(candidates)-1]
	return detectedVPN{label: p.label, version: versionOf(path, p.args)}, true
}

func walkExe(dir string, names []string, depth int, out *[]string) {
	if depth < 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			walkExe(full, names, depth-1, out)
		} else {
			lower := strings.ToLower(e.Name())
			for _, want := range names {
				if lower == strings.ToLower(want) {
					*out = append(*out, full)
					break
				}
			}
		}
	}
}

func versionOf(path string, args []string) string {
	if len(args) > 0 {
		if v := runVersion(path, args); v != "" {
			return v
		}
	}
	if v := peVersion(path); v != "" {
		return v
	}
	parent := filepath.Base(filepath.Dir(path))
	if strings.HasPrefix(parent, "v") && len(parent) > 1 {
		return parent
	}
	grandparent := filepath.Base(filepath.Dir(filepath.Dir(path)))
	if strings.HasPrefix(grandparent, "v") && len(grandparent) > 1 {
		return grandparent
	}
	return "installed"
}

func runVersion(path string, args []string) string {
	out, err := exec.Command(path, args...).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") ||
			strings.HasSuffix(lower, ".node") || strings.HasPrefix(lower, "loaded ") {
			continue
		}
		if len(line) > 60 {
			line = line[:60] + "…"
		}
		return line
	}
	return ""
}

// detectedAdapterKeys builds a set of adapterKey values from the detected list.
func detectedAdapterKeys(detected []detectedVPN) map[string]bool {
	keys := make(map[string]bool)
	for _, d := range detected {
		for _, p := range vpnProbes {
			if p.label == d.label && p.adapterKey != "" {
				keys[p.adapterKey] = true
			}
		}
	}
	return keys
}
