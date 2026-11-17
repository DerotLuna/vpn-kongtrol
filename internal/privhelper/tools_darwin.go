//go:build darwin

package privhelper

// toolSpecs: System mirrors what privops accepts (sys_unix.go and the
// safeEnv PATH); Sources are Homebrew's fixed locations (Apple silicon
// /opt/homebrew, Intel /usr/local). Apple's /bin/bash is 3.2, too old for
// wg-quick, so only a newer root-owned bash counts as a system one.
var toolSpecs = []toolSpec{
	{
		Name: "openvpn",
		System: []string{"/usr/sbin/openvpn", "/usr/local/sbin/openvpn", "/usr/bin/openvpn",
			"/opt/local/sbin/openvpn", "/opt/homebrew/sbin/openvpn"},
		Sources: []string{"/opt/homebrew/sbin/openvpn", "/opt/homebrew/opt/openvpn/sbin/openvpn",
			"/usr/local/sbin/openvpn", "/usr/local/opt/openvpn/sbin/openvpn"},
	},
	{
		Name:    "wg",
		System:  []string{"/usr/bin/wg", "/usr/local/bin/wg", "/opt/homebrew/bin/wg"},
		Sources: []string{"/opt/homebrew/bin/wg", "/usr/local/bin/wg"},
	},
	{
		Name:    "wg-quick",
		System:  []string{"/usr/bin/wg-quick", "/usr/local/bin/wg-quick", "/opt/homebrew/bin/wg-quick"},
		Sources: []string{"/opt/homebrew/bin/wg-quick", "/usr/local/bin/wg-quick"},
	},
	{
		Name:     "wireguard-go",
		System:   []string{"/usr/local/sbin/wireguard-go", "/usr/local/bin/wireguard-go", "/usr/sbin/wireguard-go", "/usr/bin/wireguard-go"},
		Sources:  []string{"/opt/homebrew/bin/wireguard-go", "/usr/local/bin/wireguard-go"},
		Requires: "wg-quick",
	},
	{
		Name:     "bash",
		System:   []string{"/usr/local/sbin/bash", "/usr/local/bin/bash"},
		Sources:  []string{"/opt/homebrew/bin/bash", "/usr/local/bin/bash"},
		Requires: "wg-quick",
	},
}
