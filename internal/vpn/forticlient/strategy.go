package forticlient

import (
	"strconv"
	"strings"
)

// Strategy is how attach-mode step 2 (automatic connect) is attempted for
// the installed FortiClient. Whatever the strategy, a failure falls back to
// step 3: open FortiClient and wait for the user (works on any version).
// See docs/forticlient-support.md and ADR-0020.
type Strategy string

const (
	// StrategyGUI64 automates the FortiClient 6.4 login form (VPN
	// dropdown, username, password, Connect) by relative coordinates.
	StrategyGUI64 Strategy = "gui-6.4"
	// StrategyFortiVPN starts `FortiVPN.exe --cli --connect --tunnel <name>`
	// and fills the native authentication dialog it opens (FortiClient
	// 7.x builds that ship FortiVPN.exe; contributed in PR #8).
	StrategyFortiVPN Strategy = "fortivpn-cli"
	// StrategyCLI is the Linux / macOS command line (forticlientsslvpn).
	StrategyCLI Strategy = "cli"
	// StrategyAttach is no automation: open FortiClient and wait.
	StrategyAttach Strategy = "attach"
)

// Support says how much a plan has been tested.
type Support string

const (
	// SupportVerified: tested end to end by a maintainer on that version.
	SupportVerified Support = "verified"
	// SupportReported: reported working by a contributor, not re-tested
	// by a maintainer.
	SupportReported Support = "reported"
	// SupportUntested: nobody has reported it yet.
	SupportUntested Support = "untested"
)

// Install describes the FortiClient found on this machine.
type Install struct {
	Path     string // FortiClient.exe (Windows) or the CLI binary
	Version  string // file ProductVersion, e.g. "6.4.10.1821"; "" = unknown
	FortiVPN string // FortiVPN.exe next to FortiClient.exe; "" = absent
}

// Plan is the strategy chosen for an install and how well it is tested.
type Plan struct {
	Strategy Strategy
	Support  Support
}

// Version is a parsed FortiClient version (major.minor.patch[.build]).
type Version struct {
	Major, Minor, Patch, Build int
}

// ParseVersion parses "6.4.10.1821" (or "7.2", "v7.0.3"). It returns
// false for an empty or non-numeric version.
func ParseVersion(s string) (Version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return Version{}, false
	}
	parts := strings.Split(s, ".")
	var n [4]int
	for i := 0; i < len(parts) && i < 4; i++ {
		v, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || v < 0 {
			return Version{}, false
		}
		n[i] = v
	}
	return Version{Major: n[0], Minor: n[1], Patch: n[2], Build: n[3]}, true
}

// verifiedGUI lists the versions on which StrategyGUI64 was tested end to
// end (major.minor.patch). Add a version only after a real connect and
// disconnect on it, and update docs/forticlient-support.md.
var verifiedGUI = map[Version]bool{
	{Major: 6, Minor: 4, Patch: 10}: true,
}

// SelectPlan picks the automatic-connect strategy for goos and install.
// It is a pure function so every row of docs/forticlient-support.md is a
// unit-test case.
//
// Windows:
//   - FortiVPN.exe present → StrategyFortiVPN (its official CLI starts the
//     tunnel; the 6.4.10 install has no FortiVPN.exe).
//   - 6.x, or version unknown → StrategyGUI64 (the layout was measured on
//     6.4.10; an unknown version keeps the behavior Kongtrol had before
//     version detection).
//   - anything else (5.x, 7.x without FortiVPN.exe, newer) → StrategyAttach:
//     the 6.4 coordinates would click a different layout.
//
// Linux / macOS: StrategyCLI.
func SelectPlan(goos string, in Install) Plan {
	if goos != "windows" {
		return Plan{Strategy: StrategyCLI, Support: SupportUntested}
	}
	if in.FortiVPN != "" {
		return Plan{Strategy: StrategyFortiVPN, Support: SupportReported}
	}
	v, ok := ParseVersion(in.Version)
	switch {
	case !ok:
		return Plan{Strategy: StrategyGUI64, Support: SupportUntested}
	case v.Major == 6:
		if verifiedGUI[Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch}] {
			return Plan{Strategy: StrategyGUI64, Support: SupportVerified}
		}
		return Plan{Strategy: StrategyGUI64, Support: SupportUntested}
	default:
		return Plan{Strategy: StrategyAttach, Support: SupportUntested}
	}
}

// fortiVPNConnectArgs are the FortiVPN.exe arguments that start a tunnel.
// Credentials are never passed on the command line (any local process can
// read another process's command line); they go into the dialog.
func fortiVPNConnectArgs(tunnel string) []string {
	return []string{"--cli", "--connect", "--tunnel", tunnel}
}

// fortiVPNDisconnectArgs are the FortiVPN.exe arguments that stop a tunnel.
func fortiVPNDisconnectArgs(tunnel string) []string {
	args := []string{"--cli", "--disconnect"}
	if tunnel != "" {
		args = append(args, "--tunnel", tunnel)
	}
	return args
}
