package nordvpn

import (
	"regexp"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// serverNameRe matches NordVPN server host names such as "us3710" or
// "us3710.nordvpn.com".
var serverNameRe = regexp.MustCompile(`^[a-z]{2}(-[a-z]{2})?\d+(\.nordvpn\.com)?$`)

// isServerName reports whether target names one specific server rather
// than a country / city / group. "United States #3710" (the app's display
// name) and "us3710" / "us3710.nordvpn.com" (host names) are servers.
func isServerName(target string) bool {
	t := strings.ToLower(strings.TrimSpace(target))
	return strings.Contains(t, "#") || serverNameRe.MatchString(t)
}

// isQuickConnect reports whether a value means "let NordVPN pick".
func isQuickConnect(v string) bool {
	t := strings.TrimSpace(v)
	return t == "" || strings.EqualFold(t, "fastest") || strings.EqualFold(t, "quick")
}

// target is the profile's server selection. Legacy holds a pre-standard
// `server:` value that is not a server name (a country / city / group such
// as "United States", "Dedicated IP", "P2P"), passed through exactly as
// before the standard fields existed.
type target struct {
	Server, Country, City, Group, Legacy string
}

// targetFrom reads the standard selection fields from AdapterConfig.Extra.
func targetFrom(extra map[string]string) target {
	t := target{
		Server:  strings.TrimSpace(extra["server"]),
		Country: strings.TrimSpace(extra["country"]),
		City:    strings.TrimSpace(extra["city"]),
		Group:   strings.TrimSpace(extra["group"]),
	}
	if isQuickConnect(t.Server) {
		t.Server = ""
	}
	if t.Server != "" && !isServerName(t.Server) && t.Group == "" && t.Country == "" && t.City == "" {
		t.Legacy, t.Server = t.Server, ""
	}
	return t
}

// windowsConnectArgs maps the target to NordVPN.exe switches (official:
// -c quick connect, -c -n "<server>", -c -g "<group>" where a group is a
// country or specialty group). NordVPN.exe takes a single -g, so group
// wins over city, city over country; passing a city to -g is best effort.
func windowsConnectArgs(t target) []string {
	switch {
	case t.Server != "":
		return []string{"-c", "-n", t.Server}
	case t.Group != "":
		return []string{"-c", "-g", t.Group}
	case t.City != "":
		return []string{"-c", "-g", t.City}
	case t.Country != "":
		return []string{"-c", "-g", t.Country}
	case t.Legacy != "":
		return []string{"-c", "-g", t.Legacy}
	default:
		return []string{"-c"}
	}
}

// windowsDisconnectArgs disconnects the NordVPN app.
func windowsDisconnectArgs() []string { return []string{"-d"} }

// underscore joins words the way the Linux CLI names them
// ("United States" -> "United_States", "Dedicated IP" -> "Dedicated_IP").
func underscore(s string) string { return strings.Join(strings.Fields(s), "_") }

// linuxConnectArgs maps the target to `nordvpn connect`: quick connect
// takes no argument; a server drops its ".nordvpn.com" suffix ("us3710");
// a group is `--group <group> [country]`; otherwise `[country] [city]`.
func linuxConnectArgs(t target) []string {
	args := []string{"connect"}
	switch {
	case t.Server != "":
		return append(args, strings.TrimSuffix(strings.ToLower(t.Server), ".nordvpn.com"))
	case t.Group != "":
		args = append(args, "--group", underscore(t.Group))
		if t.Country != "" {
			args = append(args, underscore(t.Country))
		}
		return args
	case t.Legacy != "":
		return append(args, underscore(t.Legacy))
	}
	if t.Country != "" {
		args = append(args, underscore(t.Country))
	}
	if t.City != "" {
		args = append(args, underscore(t.City))
	}
	return args
}

// linuxStatus is the subset of `nordvpn status` Kongtrol uses.
type linuxStatus struct {
	Connected bool
	Server    string // host name (e.g. us3710.nordvpn.com)
	ServerIP  string
}

// parseLinuxStatus reads `nordvpn status` output tolerantly: keys are
// matched case-insensitively, the CLI's spinner / carriage returns are
// ignored, and both current ("Hostname:", "IP:") and older ("Current
// server:", "Your new IP:", "Server IP:") key names are understood.
func parseLinuxStatus(raw string) linuxStatus {
	var st linuxStatus
	raw = strings.ReplaceAll(raw, "\r", "\n")
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimLeft(strings.TrimSpace(line), "-\\|/ ")
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "status":
			st.Connected = strings.EqualFold(val, "connected")
		case "hostname", "current server", "server":
			if st.Server == "" {
				st.Server = val
			}
		case "ip", "server ip", "your new ip":
			if st.ServerIP == "" {
				st.ServerIP = val
			}
		}
	}
	return st
}

// isNotLoggedIn reports whether CLI output says there is no NordVPN session.
func isNotLoggedIn(out string) bool {
	o := strings.ToLower(out)
	return strings.Contains(o, "not logged in") || strings.Contains(o, "please log in")
}

// app is the official NordVPN app per OS (Linux GUI launcher best effort).
var app = appattach.App{
	DisplayName:  "NordVPN",
	WindowsPaths: []string{`C:\Program Files\NordVPN\NordVPN.exe`},
	MacApp:       "NordVPN",
	LinuxCmds:    []string{"nordvpn-gui"},
}

// appFor is the NordVPN app to open in attach-mode step 3; on Windows the
// binary_path override (NordVPN.exe) is the app itself.
func appFor(binaryPath string) appattach.App {
	a := app
	if binaryPath != "" {
		a.WindowsPaths = append([]string{binaryPath}, app.WindowsPaths...)
	}
	return a
}

// label is the target for user messages ("United States", "us3710"),
// empty for quick connect.
func (t target) label() string {
	parts := make([]string, 0, 3)
	for _, v := range []string{t.Server, t.Group, t.Country, t.City, t.Legacy} {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
}
