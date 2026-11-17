//go:build linux

package privhelper

// toolSpecs: System mirrors what privops accepts (sys_unix.go and the
// safeEnv PATH); Sources are linuxbrew's fixed prefix. Distribution bash is
// >= 4, so bash is copied only when none is installed by the system.
var toolSpecs = []toolSpec{
	{
		Name: "openvpn",
		System: []string{"/usr/sbin/openvpn", "/usr/local/sbin/openvpn", "/usr/bin/openvpn",
			"/opt/local/sbin/openvpn", "/opt/homebrew/sbin/openvpn"},
		Sources: []string{"/home/linuxbrew/.linuxbrew/sbin/openvpn", "/home/linuxbrew/.linuxbrew/bin/openvpn"},
	},
	{
		Name:    "wg",
		System:  []string{"/usr/bin/wg", "/usr/local/bin/wg", "/opt/homebrew/bin/wg"},
		Sources: []string{"/home/linuxbrew/.linuxbrew/bin/wg"},
	},
	{
		Name:    "wg-quick",
		System:  []string{"/usr/bin/wg-quick", "/usr/local/bin/wg-quick", "/opt/homebrew/bin/wg-quick"},
		Sources: []string{"/home/linuxbrew/.linuxbrew/bin/wg-quick"},
	},
	{
		Name:     "bash",
		System:   []string{"/usr/local/bin/bash", "/usr/bin/bash", "/bin/bash"},
		Sources:  []string{"/home/linuxbrew/.linuxbrew/bin/bash"},
		Requires: "wg-quick",
	},
}
