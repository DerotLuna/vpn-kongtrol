package protonvpn

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

// flavor identifies which official Proton Linux CLI drives the tunnel.
type flavor int

const (
	// modernCLI is the current `protonvpn` CLI
	// (github.com/ProtonVPN/proton-vpn-cli).
	modernCLI flavor = iota
	// legacyCLI is `protonvpn-cli` v3.
	legacyCLI
)

// cliCandidates is the lookup order: the current CLI first, the legacy one
// as fallback.
var cliCandidates = []struct {
	name string
	kind flavor
}{
	{"protonvpn", modernCLI},
	{"protonvpn-cli", legacyCLI},
}

// flavorOf infers the CLI flavor from a binary path (binary_path override).
func flavorOf(path string) flavor {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if base == "protonvpn-cli" {
		return legacyCLI
	}
	return modernCLI
}

// findCLI resolves the Proton CLI: the binary_path override first, then
// `protonvpn`, then `protonvpn-cli` on PATH.
func findCLI(override string) (string, flavor, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", modernCLI, fmt.Errorf("binary_path %s: %w", override, err)
		}
		return override, flavorOf(override), nil
	}
	for _, c := range cliCandidates {
		if p, err := exec.LookPath(c.name); err == nil {
			return p, c.kind, nil
		}
	}
	return "", modernCLI, fmt.Errorf("neither protonvpn nor protonvpn-cli found on PATH — install the ProtonVPN Linux CLI and sign in first")
}

// target is the profile's server selection: an exact `server` name
// ("CH#242"), else `city`, else `country` (code such as "CH" or a name
// such as "United States"); all empty = fastest. Older profiles put a
// country code or "fastest" into `server`; targetFrom maps those.
type target struct {
	Server, Country, City string
}

// targetFrom reads the standard selection fields from AdapterConfig.Extra.
func targetFrom(extra map[string]string) target {
	t := target{
		Server:  strings.TrimSpace(extra["server"]),
		Country: strings.TrimSpace(extra["country"]),
		City:    strings.TrimSpace(extra["city"]),
	}
	switch {
	case strings.EqualFold(t.Server, "fastest"):
		t.Server = ""
	case len(t.Server) == 2 && t.Country == "": // legacy `server: CH`
		t.Country, t.Server = t.Server, ""
	}
	return t
}

// isCode reports whether a country value is a two-letter code.
func isCode(country string) bool { return len(country) == 2 }

// connectArgsFor dispatches to the flavor's argument mapping.
func connectArgsFor(kind flavor, t target, protocol string) []string {
	if kind == legacyCLI {
		return connectArgs(t, protocol)
	}
	return modernConnectArgs(t)
}

// connectArgs maps the target to protonvpn-cli v3 syntax: a server name →
// `connect <name>`, a two-letter country code → `--cc XX`, otherwise
// `--fastest` (v3 has no city selection and takes codes only).
func connectArgs(t target, protocol string) []string {
	var args []string
	switch {
	case t.Server != "":
		args = []string{"connect", t.Server}
	case isCode(t.Country):
		args = []string{"connect", "--cc", strings.ToUpper(t.Country)}
	default:
		args = []string{"connect", "--fastest"}
	}
	if protocol != "" {
		args = append(args, "-p", protocol)
	}
	return args
}

// modernConnectArgs maps the target to the current `protonvpn` CLI
// (protonvpn.com/support/use-linux-cli): `connect [server name]`,
// `connect --city <city>`, `connect --country <code or name>`, or plain
// `connect` (fastest). The modern CLI takes its protocol from its own
// settings, so `protocol` is not passed.
func modernConnectArgs(t target) []string {
	switch {
	case t.Server != "":
		return []string{"connect", t.Server}
	case t.City != "":
		return []string{"connect", "--city", t.City}
	case isCode(t.Country):
		return []string{"connect", "--country", strings.ToUpper(t.Country)}
	case t.Country != "":
		return []string{"connect", "--country", t.Country}
	default:
		return []string{"connect"}
	}
}

// connect runs the resolved CLI's connect command.
func connect(override string, t target, protocol string) error {
	bin, kind, err := findCLI(override)
	if err != nil {
		return err
	}
	return cli(bin, connectArgsFor(kind, t, protocol)...)
}

// disconnect runs the resolved CLI's disconnect command.
func disconnect(override string) error {
	bin, _, err := findCLI(override)
	if err != nil {
		return err
	}
	return cli(bin, "disconnect")
}

// version returns the first line of `<cli> --version`.
func version(override string) string {
	bin, _, err := findCLI(override)
	if err != nil {
		return ""
	}
	out, _ := cliOutput(bin, "--version")
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(line)
}

// connectionState reports whether the tunnel is up plus whatever the status
// output exposes. The legacy CLI has a documented `status` (JSON in v3.10+);
// the modern CLI's status command is undocumented, so its output is parsed
// tolerantly and, when it says nothing conclusive, the Proton network
// interface (proton0 / pvpn*) decides.
func connectionState(override string) (connected bool, serverIP, assignedIP string, err error) {
	bin, kind, err := findCLI(override)
	if err != nil {
		return false, "", "", err
	}
	if kind == legacyCLI {
		raw, err := legacyStatusOutput(bin)
		if err != nil {
			return false, "", "", err
		}
		connected, serverIP, assignedIP = parseStatus(raw)
		return connected, serverIP, assignedIP, nil
	}

	raw, _ := cliOutput(bin, "status")
	known, up := parseModernStatus(raw)
	_, serverIP, assignedIP = parseStatus(raw)
	if !known {
		up = false
		if t, ok := appattach.Find(appattach.Interfaces(), matcher); ok {
			ip := t.IP
			up = true
			if assignedIP == "" && ip != nil {
				assignedIP = ip.String()
			}
		}
	}
	return up, serverIP, assignedIP, nil
}

// legacyStatusOutput returns `protonvpn-cli status`, JSON when supported.
func legacyStatusOutput(bin string) (string, error) {
	out, err := cliOutput(bin, "status", "--json")
	if err == nil && strings.HasPrefix(strings.TrimSpace(out), "{") {
		return out, nil
	}
	return cliOutput(bin, "status")
}

// parseModernStatus reads the (undocumented) `protonvpn status` output.
// known is false when the output does not clearly state the connection
// state (command missing, unexpected format), so callers fall back to
// interface detection.
func parseModernStatus(raw string) (known, connected bool) {
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r", "\n"), "\n") {
		l := strings.ToLower(strings.TrimSpace(line))
		switch {
		case l == "":
			continue
		case strings.HasPrefix(l, "status:"):
			v := strings.TrimSpace(strings.TrimPrefix(l, "status:"))
			return true, v == "connected"
		case strings.Contains(l, "not connected"), strings.HasPrefix(l, "disconnected"),
			strings.Contains(l, "no active connection"):
			return true, false
		case strings.HasPrefix(l, "connected to"), strings.HasPrefix(l, "connected"):
			return true, true
		}
	}
	return false, false
}

// parseStatus extracts the connection state from protonvpn-cli status output.
// Handles both JSON (v3.10+) and human-readable (older) formats.
func parseStatus(raw string) (connected bool, serverIP, assignedIP string) {
	raw = strings.TrimSpace(raw)

	// JSON format: {"Status":"Connected","ServerIP":"...","IP":"..."}
	if strings.HasPrefix(raw, "{") {
		connected = strings.Contains(raw, `"Status":"Connected"`)
		serverIP = extractJSON(raw, "ServerIP")
		assignedIP = extractJSON(raw, "IP")
		return
	}

	// Human-readable: look for "Status:" and "IP:" lines.
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "Status:")))
			connected = v == "connected"
		}
		if strings.HasPrefix(line, "Server IP:") {
			serverIP = strings.TrimSpace(strings.TrimPrefix(line, "Server IP:"))
		}
		if strings.HasPrefix(line, "IP:") {
			assignedIP = strings.TrimSpace(strings.TrimPrefix(line, "IP:"))
		}
	}
	return
}

func extractJSON(s, key string) string {
	needle := `"` + key + `":"`
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func cli(bin string, args ...string) error {
	out, err := cliOutput(bin, args...)
	if err != nil {
		return fmt.Errorf("%s %v: %w\n%s", filepath.Base(bin), args, err, out)
	}
	return nil
}

func cliOutput(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	vpn.HideChildWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
