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
	AnimateLogo(w.t("banner.subtitle"))
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
		profile, vpnNode, err := w.collectProfile()
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
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.status")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.up")))
	fmt.Println("  " + paint(cPrompt, w.t("nextsteps.dashboard")))
	fmt.Println()
	return nil
}

// ── profile collection ────────────────────────────────────────────────────────

func (w *wizard) collectProfile() (string, *yaml.Node, error) {
	SectionHeader(w.t("section.new_profile"))

	name := w.prompt(w.t("collect.profile_name"), "")
	if name == "" {
		return "", nil, fmt.Errorf("%s", w.t("collect.profile_name_empty"))
	}
	name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	fmt.Println(paint(cDim, w.t("collect.adapter_line1")))
	fmt.Println(paint(cDim, w.t("collect.adapter_line2")))
	adapterType := w.prompt(w.t("collect.type"), "openvpn")

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

// vpnProbe describes how to locate and version-check one VPN client.
type vpnProbe struct {
	label    string
	// binaries are checked via exec.LookPath first, then os.Stat for absolute paths.
	binaries []string
	// winGlobs are filepath.Glob patterns tried after binaries fail.
	// Useful for versioned install dirs like ProtonVPN's v4.x.y subdirectories.
	// The latest lexicographic match wins (higher version numbers sort last).
	winGlobs []string
	// args are passed to the binary to retrieve a version string.
	// Leave empty to skip the version command (GUI-only apps).
	args []string
}

var vpnProbes = []vpnProbe{
	{
		label: "FortiClient",
		binaries: []string{
			"fortivpn", "forticlientsslvpn", "FortiClient",
			`C:\Program Files\Fortinet\FortiClient\FortiClient.exe`,
			`C:\Program Files (x86)\Fortinet\FortiClient\FortiClient.exe`,
			`C:\Program Files\Fortinet\FortiClient EMS\FortiClient.exe`,
		},
		winGlobs: []string{
			`C:\Program Files\Fortinet\FortiClient\*\FortiClient.exe`,
			`C:\Program Files (x86)\Fortinet\FortiClient\*\FortiClient.exe`,
		},
		// FortiClient is an Electron GUI app — --version outputs JS module noise.
		// Version is read from PE metadata via peVersion() instead.
		args: []string{},
	},
	{
		label: "OpenVPN",
		binaries: []string{
			"openvpn",
			`C:\Program Files\OpenVPN\bin\openvpn.exe`,
			`C:\Program Files (x86)\OpenVPN\bin\openvpn.exe`,
			`C:\Program Files\OpenVPN Connect\OpenVPNConnect.exe`,
		},
		args: []string{"--version"},
	},
	{
		label: "ProtonVPN",
		binaries: []string{
			"protonvpn-cli",
			`C:\Program Files\Proton\VPN\ProtonVPN.Launcher.exe`,
			`C:\Program Files (x86)\Proton Technologies\ProtonVPN\ProtonVPN.exe`,
		},
		// ProtonVPN installs into versioned subdirs: v4.x.y/ProtonVPN.Client.exe
		winGlobs: []string{
			`C:\Program Files\Proton\VPN\v*\ProtonVPN.Client.exe`,
			`C:\Program Files (x86)\Proton\VPN\v*\ProtonVPN.Client.exe`,
		},
		// GUI app — no reliable --version flag; version extracted from dir name instead.
		args: []string{},
	},
	{
		label: "Cisco AnyConnect",
		binaries: []string{
			"vpn", "vpncli",
			"/opt/cisco/anyconnect/bin/vpn",
			`C:\Program Files (x86)\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`,
			`C:\Program Files\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`,
			`C:\Program Files (x86)\Cisco\Cisco Secure Client\vpncli.exe`,
			`C:\Program Files\Cisco\Cisco Secure Client\vpncli.exe`,
		},
		args: []string{"-v"},
	},
	{
		label: "WireGuard",
		binaries: []string{
			"wg", "wg-quick", "wireguard",
			`C:\Program Files\WireGuard\wg.exe`,
			`C:\Program Files\WireGuard\wireguard.exe`,
		},
		args: []string{"--version"},
	},
	{
		label: "GlobalProtect",
		binaries: []string{
			"globalprotect", "pangpcrypt",
			"/opt/paloaltonetworks/globalprotect/pangpcrypt",
			`C:\Program Files\Palo Alto Networks\GlobalProtect\PanGPA.exe`,
			`C:\Program Files\Palo Alto Networks\GlobalProtect\pangpcrypt.exe`,
		},
		args: []string{"--version"},
	},
	{
		label: "Tailscale",
		binaries: []string{
			"tailscale",
			`C:\Program Files\Tailscale\tailscale.exe`,
		},
		args: []string{"version"},
	},
	{
		label: "Cloudflare WARP",
		binaries: []string{
			"warp-cli",
			`C:\Program Files\Cloudflare\Cloudflare WARP\warp-cli.exe`,
		},
		args: []string{"--version"},
	},
	{
		label: "TunnelBear",
		binaries: []string{
			"tunnelbear",
			`C:\Program Files (x86)\TunnelBear\TunnelBear.exe`,
			`C:\Program Files\TunnelBear\TunnelBear.exe`,
			"/Applications/TunnelBear.app/Contents/MacOS/TunnelBear",
		},
		winGlobs: []string{
			`C:\Program Files (x86)\TunnelBear\*\TunnelBear.exe`,
			`C:\Program Files\TunnelBear\*\TunnelBear.exe`,
		},
		args: []string{"--version"},
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

// resolveProbe tries to locate the VPN binary for p and returns its version.
func resolveProbe(p vpnProbe) (detectedVPN, bool) {
	// 1. PATH lookup + absolute-path stat for each binary entry.
	for _, bin := range p.binaries {
		path, err := exec.LookPath(bin)
		if err != nil {
			if _, statErr := os.Stat(bin); statErr != nil {
				continue
			}
			path = bin
		}
		ver := runVersion(path, p.args)
		if ver == "installed" {
			// Try reading version from Windows PE metadata (works for GUI apps
			// that don't expose a --version flag, e.g. FortiClient, TunnelBear).
			if v := peVersion(path); v != "" {
				ver = v
			}
		}
		if ver == "installed" {
			// Last resort: extract version from highest "v*" sibling directory
			// (e.g. ProtonVPN installs into …\Proton\VPN\v4.3.14\).
			parent := filepath.Dir(path)
			if dirs, globErr := filepath.Glob(filepath.Join(parent, "v*")); globErr == nil && len(dirs) > 0 {
				sort.Strings(dirs)
				ver = filepath.Base(dirs[len(dirs)-1])
			}
		}
		return detectedVPN{label: p.label, version: ver}, true
	}

	// 2. Glob patterns — useful for versioned install directories.
	for _, pattern := range p.winGlobs {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			continue
		}
		// Sort ascending so the last entry is the highest version.
		sort.Strings(matches)
		path := matches[len(matches)-1]

		ver := runVersion(path, p.args)
		// If the binary gave no version, try extracting it from the parent dir name
		// (e.g. "v4.3.14" from "…\Proton\VPN\v4.3.14\ProtonVPN.Client.exe").
		if ver == "installed" {
			if dir := filepath.Base(filepath.Dir(path)); strings.HasPrefix(dir, "v") {
				ver = dir
			}
		}
		return detectedVPN{label: p.label, version: ver}, true
	}

	return detectedVPN{}, false
}

// runVersion executes path with args and returns the first meaningful version
// line from stdout. Falls back to "installed" when args is empty, the command
// fails, or the output doesn't look like a version string.
func runVersion(path string, args []string) string {
	if len(args) == 0 {
		return "installed"
	}
	out, err := exec.Command(path, args...).Output()
	if err != nil || len(out) == 0 {
		return "installed"
	}
	// Scan lines for the first one that looks like a version string.
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip lines that are just executable filenames or other noise.
		lower := strings.ToLower(line)
		if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") {
			continue
		}
		if len(line) > 60 {
			line = line[:60] + "…"
		}
		return line
	}
	return "installed"
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
