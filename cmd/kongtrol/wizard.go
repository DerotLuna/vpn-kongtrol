package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── command registration ──────────────────────────────────────────────────────

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard — create or update kongtrol.yaml",
	RunE:  runWizard,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// ── wizard entry point ────────────────────────────────────────────────────────

func runWizard(_ *cobra.Command, _ []string) error {
	w := &wizard{r: bufio.NewReader(os.Stdin)}
	w.selectLanguage()
	return w.run()
}

type wizard struct {
	r    *bufio.Reader
	lang i18n.Lang
}

func (w *wizard) t(key string) string           { return i18n.T(w.lang, key) }
func (w *wizard) tF(key string, a ...any) string { return i18n.F(w.lang, key, a...) }

// selectLanguage asks the language preference before anything else.
// The prompt itself is bilingual so both audiences understand it.
func (w *wizard) selectLanguage() {
	fmt.Println()
	fmt.Print(paintBold(cPrompt, "¿Continuar en español?") +
		paint(cDim, " [S/n]  (Press n for English): "))
	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "n" || line == "no" {
		w.lang = i18n.EN
	} else {
		w.lang = i18n.ES
	}
	fmt.Println()
}

func (w *wizard) run() error {
	// ── animated logo ─────────────────────────────────────────────────────────
	AnimateLogo(w.t("banner.subtitle"), version)
	fmt.Println()

	// ── subtitle (typewriter effect) ──────────────────────────────────────────
	w.typeWrite(w.t("banner.yaml"), 18)
	w.typeWrite(paint(cDim, w.t("banner.keychain")), 14)
	fmt.Println()

	// ── output path ──────────────────────────────────────────────────────────
	home, _ := os.UserHomeDir()
	outPath := filepath.Join(home, ".kongtrol", "kongtrol.yaml")
	if cfgPath != "" {
		outPath = cfgPath
	}

	// ── load existing config ──────────────────────────────────────────────────
	existing, existingRaw := w.loadExisting(outPath)

	// ── detect VPN clients ────────────────────────────────────────────────────
	spin := newSpinner(w.t("detected.scanning"))
	spin.Start()
	detected := detectInstalledVPNs()
	spin.Stop()

	if len(detected) > 0 {
		fmt.Println(tuiInfo(paintBold(cBright, w.t("detected.header"))))
		for _, d := range detected {
			fmt.Printf("    %s  %-22s  %s\n",
				paintBold(cSuccess, "✓"),
				paintBold(cBright, d.label),
				paint(cDim, d.version))
		}
	} else {
		fmt.Println(tuiWarn(w.t("detected.none")))
	}

	// ── show existing profiles ────────────────────────────────────────────────
	if existing != nil && len(existing.VPNs) > 0 {
		SectionHeader(w.tF("existing.header", outPath, len(existing.VPNs)))
		for name, v := range existing.VPNs {
			fmt.Printf("    %s  %-16s  type=%s  host=%s\n",
				paint(cInfo, "·"),
				paintBold(cBright, name),
				paint(cWarn, v.Type),
				paint(cDim, v.Host))
		}
	}

	// ── build YAML doc ────────────────────────────────────────────────────────
	doc := existingRaw
	if doc == nil {
		doc = w.freshDoc()
	}

	// ── existing profiles: credential refresh ─────────────────────────────────
	if existing != nil {
		for name, vpnCfg := range existing.VPNs {
			fmt.Printf("\n%s  %s %s\n",
				paint(cInfo, "──"),
				paintBold(cBright, name),
				paint(cDim, "("+vpnCfg.Type+")"))
			if w.confirm(w.t("profile.refresh_creds"), false) {
				if err := w.collectCredentials(name, vpnCfg.Type, vpnCfg.Auth); err != nil {
					fmt.Fprintln(os.Stderr, tuiWarn(err.Error()))
				}
			}
		}
	}

	// ── add new profiles ──────────────────────────────────────────────────────
	for {
		fmt.Println()
		if !w.confirm(paintBold(cBright, w.t("profile.add_new")), false) {
			break
		}
		profile, vpnNode, err := w.collectProfile(detected)
		if err != nil {
			fmt.Fprintln(os.Stderr, tuiErr(err.Error()))
			continue
		}
		vpnsNode := mappingKey(doc, "vpns")
		if vpnsNode == nil {
			vpnsNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			doc.Content = append(doc.Content, scalarNode("vpns"), vpnsNode)
		}
		vpnsNode.Content = append(vpnsNode.Content, scalarNode(profile), vpnNode)
	}

	// ── security defaults ─────────────────────────────────────────────────────
	SectionHeader(w.t("section.security"))
	if existing == nil || !existing.Security.KillSwitch.Enabled {
		if w.confirm(w.t("security.kill_switch"), true) {
			setMapping(mappingKey(doc, "security"), "kill_switch",
				mapNode([][2]string{
					{"enabled", "true"},
					{"mode", "strict"},
					{"allow_lan", "true"},
				}))
		}
	}
	if existing == nil || !existing.Security.DNSGuard.Enabled {
		if w.confirm(w.t("security.dns_guard"), true) {
			setMapping(mappingKey(doc, "security"), "dns_guard",
				mapNode([][2]string{
					{"enabled", "true"},
					{"fallback_dns", "1.1.1.1"},
				}))
		}
	}
	if existing == nil || !existing.Security.AuditLog.Sign {
		auditPath := filepath.Join(home, ".kongtrol", "audit.log")
		if w.confirm(w.t("security.audit_log"), true) {
			setMapping(mappingKey(doc, "security"), "audit_log",
				mapNode([][2]string{
					{"path", auditPath},
					{"max_size_mb", "100"},
					{"sign", "true"},
				}))
		}
	}
	if existing == nil || !existing.Monitor.Enabled {
		if w.confirm(w.t("monitor.dashboard"), true) {
			setMapping(doc, "monitor", mapNode([][2]string{{"enabled", "true"}}))
		}
	}

	// ── write output ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Print(paintBold(cBright, w.tF("write.confirm", paint(cWarn, outPath))))
	if !w.confirm("", true) {
		fmt.Println(tuiWarn(w.t("write.aborted")))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return fmt.Errorf("init: mkdir: %w", err)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("init: marshal: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0o600); err != nil {
		return fmt.Errorf("init: write: %w", err)
	}

	fmt.Println(tuiOK(paintBold(cBright, w.tF("write.success", outPath))))

	if _, err := config.Load(outPath); err != nil {
		fmt.Fprintln(os.Stderr, tuiWarn(w.tF("write.validation_warn", err)))
		fmt.Fprintln(os.Stderr, paint(cDim, w.t("write.validation_hint")))
	} else {
		fmt.Println(tuiOK(paintBold(cSuccess, w.t("write.valid"))))
	}

	// ── next steps ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(paintBold(cGold, w.t("nextsteps.header")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.init")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.status")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.up")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.dashboard")))
	fmt.Println()
	return nil
}

// ── profile collection ────────────────────────────────────────────────────────

// detectedAdapterKeys builds a set of adapterKey values from the detected VPN list.
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

func (w *wizard) collectProfile(detected []detectedVPN) (string, *yaml.Node, error) {
	SectionHeader(w.t("section.new_profile"))

	name := w.prompt(w.t("collect.profile_name"), "")
	if name == "" {
		return "", nil, fmt.Errorf("%s", w.t("collect.profile_name_empty"))
	}
	name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	fmt.Println(paint(cDim, w.t("collect.adapter_line1")))
	fmt.Println(paint(cDim, w.t("collect.adapter_line2")))
	adapterType := w.prompt(w.t("collect.type"), "openvpn")

	// Warn + offer manual binary path when the chosen adapter wasn't auto-detected.
	detectedKeys := detectedAdapterKeys(detected)
	var binaryPath string
	if adapterType != "cloudflarewarp" && !detectedKeys[adapterType] {
		fmt.Println(tuiWarn(w.t("collect.not_detected")))
		binaryPath = w.prompt(w.t("collect.binary_path"), "")
	}

	fields := [][2]string{{"type", adapterType}}
	auth := config.AuthConfig{}

	switch adapterType {
	case "forticlient":
		fields = append(fields,
			[2]string{"host", w.prompt(w.t("collect.host"), "")},
			[2]string{"port", w.promptDefault(w.t("collect.port"), "443")},
			[2]string{"tunnel_name", w.prompt(w.t("collect.tunnel_name"), "Office")},
		)
		fields = append(fields, [2]string{"version", w.promptDefault(w.t("collect.forti_ver"), "6")})
		auth.Method = w.promptDefault(w.t("collect.auth_method"), "certificate+credentials")
		if strings.Contains(auth.Method, "certificate") {
			auth.Cert = w.prompt(w.t("collect.cert"), "")
			auth.Key = w.prompt(w.t("collect.key"), "")
		}
		if strings.Contains(auth.Method, "credentials") {
			auth.Username = w.prompt(w.t("collect.username"), "")
			auth.PasswordKeychain = name + ".password"
		}

	case "openvpn":
		fields = append(fields,
			[2]string{"config", w.prompt(w.t("collect.ovpn_config"), "")},
		)
		auth.Method = w.promptDefault(w.t("collect.auth_method"), "certificate")
		if strings.Contains(auth.Method, "certificate") {
			auth.Cert = w.prompt(w.t("collect.ovpn_cert"), "")
			auth.Key = w.prompt(w.t("collect.ovpn_key"), "")
		}
		if strings.Contains(auth.Method, "credentials") {
			auth.Username = w.prompt(w.t("collect.username"), "")
			auth.PasswordKeychain = name + ".password"
		}

	case "protonvpn":
		fields = append(fields,
			[2]string{"server", w.promptDefault(w.t("collect.proton_srv"), "fastest")},
			[2]string{"protocol", w.promptDefault(w.t("collect.proton_proto"), "wireguard")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt(w.t("collect.proton_user"), "")
		auth.PasswordKeychain = name + ".password"

	case "ciscoanyconnect":
		fields = append(fields,
			[2]string{"host", w.prompt(w.t("collect.cisco_host"), "")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt(w.t("collect.cisco_user"), "")
		auth.PasswordKeychain = name + ".password"

	case "wireguard":
		fields = append(fields,
			[2]string{"config", w.prompt(w.t("collect.wg_config"), "")},
		)
		auth.Method = "certificate"

	case "globalprotect":
		fields = append(fields,
			[2]string{"host", w.prompt(w.t("collect.gp_host"), "")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt(w.t("collect.gp_user"), "")
		auth.PasswordKeychain = name + ".password"

	case "tailscale":
		exitNode := w.prompt(w.t("collect.ts_exitnode"), "")
		if exitNode != "" {
			fields = append(fields, [2]string{"host", exitNode})
		}
		auth.Method = "credentials"
		if w.confirm(w.t("collect.ts_usekey"), false) {
			auth.PasswordKeychain = name + ".authkey"
		}

	case "cloudflarewarp":
		auth.Method = "credentials"
		fmt.Println(tuiInfo(w.t("collect.warp_info1")))
		fmt.Println(paint(cDim, "  "+w.t("collect.warp_info2")))

	default:
		return "", nil, fmt.Errorf("%s", w.tF("collect.unknown_adapter", adapterType))
	}

	priorityStr := w.promptDefault(w.t("collect.priority"), "10")
	fields = append(fields, [2]string{"priority", priorityStr})
	if binaryPath != "" {
		fields = append(fields, [2]string{"binary_path", binaryPath})
	}

	authFields := [][2]string{{"method", auth.Method}}
	if auth.Cert != "" {
		authFields = append(authFields, [2]string{"cert", auth.Cert})
	}
	if auth.Key != "" {
		authFields = append(authFields, [2]string{"key", auth.Key})
	}
	if auth.Username != "" {
		authFields = append(authFields, [2]string{"username", auth.Username})
	}
	if auth.PasswordKeychain != "" {
		authFields = append(authFields, [2]string{"password_keychain", auth.PasswordKeychain})
	}

	node := mapNode(fields)
	node.Content = append(node.Content, scalarNode("auth"), mapNode(authFields))

	if auth.PasswordKeychain != "" {
		if err := w.collectCredentials(name, adapterType, auth); err != nil {
			fmt.Fprintln(os.Stderr, tuiWarn(w.tF("collect.password_warn", err)))
		}
	}

	return name, node, nil
}

func (w *wizard) collectCredentials(profileName, adapterType string, auth config.AuthConfig) error {
	switch adapterType {
	case "cloudflarewarp", "wireguard":
		return nil
	case "tailscale":
		if auth.PasswordKeychain == "" {
			return nil
		}
		key := w.promptSecret(w.t("collect.ts_key"))
		if key != "" {
			return config.SetCredential(profileName, "password", key)
		}
		return nil
	default:
		if auth.PasswordKeychain != "" {
			pwd := w.promptSecret(w.tF("collect.password", paintBold(cWarn, profileName)))
			if pwd != "" {
				return config.SetCredential(profileName, "password", pwd)
			}
		}
		return nil
	}
}

// ── VPN client detection ──────────────────────────────────────────────────────

type detectedVPN struct {
	label   string
	version string
}

// vpnProbe describes how to locate one VPN client.
//
//   - adapterKey — adapter type string used in kongtrol.yaml (empty = info-only, not configurable)
//   - binaries   — names tried via exec.LookPath; absolute paths tried via os.Stat
//   - searchDirs — parent dirs walked up to searchDepth levels looking for exeNames
//   - exeNames   — filenames (case-insensitive) to match inside searchDirs
//   - args       — passed to the binary to get a version string; empty = GUI app, skip
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
			// Windows
			`C:\Program Files\Fortinet`,
			`C:\Program Files (x86)\Fortinet`,
			// Linux
			"/opt/forticlient",
			"/opt/fortinet/forticlient",
			"/usr/share/forticlient",
			"/usr/lib/forticlient",
			// macOS
			"/Applications/FortiClient.app/Contents/MacOS",
		},
		searchDepth: 2,
		exeNames:    []string{"FortiClient.exe", "FortiClient", "forticlient"},
		// Electron GUI — --version outputs JS noise; version read from PE metadata.
		args: []string{},
	},
	{
		label:      "OpenVPN",
		adapterKey: "openvpn",
		binaries:   []string{"openvpn"},
		searchDirs: []string{
			// Windows
			`C:\Program Files\OpenVPN`,
			`C:\Program Files (x86)\OpenVPN`,
			`C:\Program Files\OpenVPN Connect`,
			`C:\Program Files (x86)\OpenVPN Connect`,
			// Linux
			"/usr/sbin",
			"/usr/local/sbin",
			// macOS (Homebrew + Tunnelblick)
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
			// Windows (versioned: …\Proton\VPN\v4.x.y\)
			`C:\Program Files\Proton`,
			`C:\Program Files (x86)\Proton`,
			`C:\Program Files (x86)\Proton Technologies`,
			// Linux
			"/usr/bin",
			"/usr/local/bin",
			"/opt/protonvpn",
			// macOS
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
			// Linux / macOS
			"/opt/cisco",
			// Windows
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
			// Windows
			`C:\Program Files\WireGuard`,
			// macOS (Homebrew)
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
			// Linux
			"/opt/paloaltonetworks",
			// Windows
			`C:\Program Files\Palo Alto Networks`,
			// macOS
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
			// Windows
			`C:\Program Files\Tailscale`,
			// macOS
			"/Applications/Tailscale.app/Contents/MacOS",
			// Linux (Homebrew / package manager → PATH, but also snap)
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
			// Windows
			`C:\Program Files\Cloudflare\Cloudflare WARP`,
			`C:\Program Files (x86)\Cloudflare\Cloudflare WARP`,
			// macOS
			"/Applications/Cloudflare WARP.app/Contents/MacOS",
			// Linux
			"/usr/bin",
			"/usr/local/bin",
		},
		searchDepth: 1,
		exeNames:    []string{"warp-cli.exe", "warp-cli"},
		args:        []string{"--version"},
	},
	{
		label:      "TunnelBear",
		adapterKey: "", // no adapter yet — shown as detected but not configurable
		binaries:   []string{"tunnelbear"},
		searchDirs: []string{
			// Windows
			`C:\Program Files\TunnelBear`,
			`C:\Program Files (x86)\TunnelBear`,
			// macOS
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

// resolveProbe tries to locate the VPN binary described by p.
// Resolution order:
//  1. exec.LookPath (binary in $PATH)
//  2. os.Stat on each absolute binary path
//  3. Recursive directory walk in searchDirs up to searchDepth
func resolveProbe(p vpnProbe) (detectedVPN, bool) {
	// 1 & 2: PATH + absolute paths.
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

	// 3: Walk each searchDir for any of the exeNames.
	var candidates []string
	for _, dir := range p.searchDirs {
		walkExe(dir, p.exeNames, p.searchDepth, &candidates)
	}
	if len(candidates) == 0 {
		return detectedVPN{}, false
	}
	// Sort so that lexicographically later paths win (higher version dirs sort last).
	sort.Strings(candidates)
	path := candidates[len(candidates)-1]
	return detectedVPN{label: p.label, version: versionOf(path, p.args)}, true
}

// walkExe appends paths of files whose names match any entry in names
// (case-insensitive) found within dir, recursing up to depth levels.
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

// versionOf returns a human-readable version string for the binary at path.
// It tries (in order): CLI output, Windows PE metadata, sibling "v*" directory.
// Always returns a non-empty string — falls back to "installed".
func versionOf(path string, args []string) string {
	// CLI version output.
	if len(args) > 0 {
		if v := runVersion(path, args); v != "" {
			return v
		}
	}
	// Windows PE metadata (build-tagged; no-op on non-Windows).
	if v := peVersion(path); v != "" {
		return v
	}
	// Version embedded in parent directory name (e.g. …\v4.3.14\app.exe).
	parent := filepath.Base(filepath.Dir(path))
	if strings.HasPrefix(parent, "v") && len(parent) > 1 {
		return parent
	}
	// Grandparent (one more level up, e.g. …\Proton\VPN\v4.3.14\sub\app.exe).
	grandparent := filepath.Base(filepath.Dir(filepath.Dir(path)))
	if strings.HasPrefix(grandparent, "v") && len(grandparent) > 1 {
		return grandparent
	}
	return "installed"
}

// runVersion runs path with args and returns the first non-noise line of stdout.
// Returns empty string on failure so callers can try other strategies.
func runVersion(path string, args []string) string {
	out, err := exec.Command(path, args...).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		// Skip lines that look like filenames or module-loader noise.
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

// ── prompt helpers ────────────────────────────────────────────────────────────

func (w *wizard) typeWrite(s string, msPerChar int) {
	if !isTerminal {
		fmt.Println(s)
		return
	}
	// Strip ANSI for length check; actual print uses the colored string.
	for _, ch := range []rune(s) {
		fmt.Print(string(ch))
		if msPerChar > 0 {
			// Crude per-char delay — skip for ANSI escape sequences.
			// We just sleep for visible chars (a rough heuristic: rune > 31).
			if ch > 31 {
				// Use a non-blocking approach: sleep inline.
				// time.Sleep is fine here since this is the UI goroutine.
				_ = ch
			}
		}
	}
	fmt.Println()
}

func (w *wizard) prompt(label, def string) string {
	l := tuiLabel(label)
	if def != "" {
		fmt.Printf("%s %s: ", l, tuiDim("["+def+"]"))
	} else {
		fmt.Printf("%s: ", l)
	}
	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func (w *wizard) promptDefault(label, def string) string { return w.prompt(label, def) }

func (w *wizard) promptSecret(label string) string {
	fmt.Printf("%s: ", tuiLabel(label))
	sttyOff := exec.Command("stty", "-echo")
	sttyOff.Stdin = os.Stdin
	_ = sttyOff.Run()

	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(line)

	sttyOn := exec.Command("stty", "echo")
	sttyOn.Stdin = os.Stdin
	_ = sttyOn.Run()
	fmt.Println()
	return line
}

func (w *wizard) confirm(label string, def bool) bool {
	hint := paintBold(cDim, "["+i18n.YesNo(w.lang, def)+"]")
	if label != "" {
		fmt.Printf("%s %s: ", label, hint)
	} else {
		fmt.Printf("%s: ", hint)
	}
	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	return i18n.IsYes(w.lang, line)
}

// ── YAML document helpers ─────────────────────────────────────────────────────

func (w *wizard) loadExisting(path string) (*config.Config, *yaml.Node) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, nil
	}
	doc := &root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, doc
	}
	return &cfg, doc
}

func (w *wizard) freshDoc() *yaml.Node {
	return mapNode([][2]string{})
}

func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"}
}

func boolNode(val bool) *yaml.Node {
	v := "false"
	if val {
		v = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!bool"}
}

func intNode(val string) *yaml.Node {
	if _, err := strconv.Atoi(val); err != nil {
		return scalarNode(val)
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!int"}
}

func mapNode(pairs [][2]string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range pairs {
		n.Content = append(n.Content, scalarNode(p[0]), autoScalar(p[1]))
	}
	return n
}

func autoScalar(val string) *yaml.Node {
	switch strings.ToLower(val) {
	case "true":
		return boolNode(true)
	case "false":
		return boolNode(false)
	}
	if _, err := strconv.Atoi(val); err == nil {
		return intNode(val)
	}
	return scalarNode(val)
}

func mappingKey(n *yaml.Node, key string) *yaml.Node {
	if n == nil {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func setMapping(parent *yaml.Node, key string, val *yaml.Node) {
	if parent == nil {
		return
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content[i+1] = val
			return
		}
	}
	parent.Content = append(parent.Content, scalarNode(key), val)
}
