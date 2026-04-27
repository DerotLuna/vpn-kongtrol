package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
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
	return w.run()
}

type wizard struct {
	r *bufio.Reader
}

func (w *wizard) run() error {
	w.banner()

	// Determine output path.
	home, _ := os.UserHomeDir()
	outPath := filepath.Join(home, ".kongtrol", "kongtrol.yaml")
	if cfgPath != "" {
		outPath = cfgPath
	}

	// Load existing config (if any) so we can preserve already-configured profiles.
	existing, existingRaw := w.loadExisting(outPath)

	// Detect installed VPN clients.
	detected := detectInstalledVPNs()
	if len(detected) > 0 {
		fmt.Println("\nDetected VPN clients on this system:")
		for _, d := range detected {
			fmt.Printf("  ✓ %-20s  (%s)\n", d.label, d.version)
		}
	} else {
		fmt.Println("\nNo VPN clients auto-detected (they may still work if installed elsewhere).")
	}

	// Show existing profiles.
	if existing != nil && len(existing.VPNs) > 0 {
		fmt.Printf("\nExisting config found at %s with %d profile(s):\n", outPath, len(existing.VPNs))
		for name, v := range existing.VPNs {
			fmt.Printf("  • %-16s  type=%-16s  host=%s\n", name, v.Type, v.Host)
		}
	}

	// Build merged YAML document (preserves comments in existing raw YAML when
	// no structural changes are made; falls back to fresh generation when profiles
	// are added).
	doc := existingRaw
	if doc == nil {
		doc = w.freshDoc()
	}

	// ── existing profiles: offer credential refresh ───────────────────────────
	if existing != nil {
		for name, vpnCfg := range existing.VPNs {
			fmt.Printf("\n── Profile: %s (%s) ──\n", name, vpnCfg.Type)
			if w.confirm("  Refresh / update credentials for this profile?", false) {
				if err := w.collectCredentials(name, vpnCfg.Type, vpnCfg.Auth); err != nil {
					fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
				}
			}
		}
	}

	// ── add new profiles ──────────────────────────────────────────────────────
	for {
		fmt.Println()
		if !w.confirm("Add a new VPN profile?", false) {
			break
		}
		profile, vpnNode, err := w.collectProfile()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}

		// Inject into the YAML document.
		vpnsNode := mappingKey(doc, "vpns")
		if vpnsNode == nil {
			// Create vpns mapping if missing.
			vpnsNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			doc.Content = append(doc.Content,
				scalarNode("vpns"),
				vpnsNode,
			)
		}
		vpnsNode.Content = append(vpnsNode.Content,
			scalarNode(profile),
			vpnNode,
		)
	}

	// ── security defaults (skip if already configured) ────────────────────────
	if existing == nil || !existing.Security.KillSwitch.Enabled {
		fmt.Println()
		if w.confirm("Enable kill switch? (blocks all traffic if VPN drops)", true) {
			setMapping(mappingKey(doc, "security"), "kill_switch",
				mapNode([][2]string{
					{"enabled", "true"},
					{"mode", "strict"},
					{"allow_lan", "true"},
				}))
		}
	}
	if existing == nil || !existing.Security.DNSGuard.Enabled {
		if w.confirm("Enable DNS guard? (prevents DNS leaks)", true) {
			setMapping(mappingKey(doc, "security"), "dns_guard",
				mapNode([][2]string{
					{"enabled", "true"},
					{"fallback_dns", "1.1.1.1"},
				}))
		}
	}
	if existing == nil || !existing.Security.AuditLog.Sign {
		auditPath := filepath.Join(home, ".kongtrol", "audit.log")
		if w.confirm("Enable signed audit log?", true) {
			setMapping(mappingKey(doc, "security"), "audit_log",
				mapNode([][2]string{
					{"path", auditPath},
					{"max_size_mb", "100"},
					{"sign", "true"},
				}))
		}
	}

	// ── monitor defaults ──────────────────────────────────────────────────────
	if existing == nil || !existing.Monitor.Enabled {
		if w.confirm("Enable web dashboard? (http://127.0.0.1:9741)", true) {
			setMapping(doc, "monitor",
				mapNode([][2]string{
					{"enabled", "true"},
				}))
		}
	}

	// ── write output ──────────────────────────────────────────────────────────
	fmt.Printf("\nWrite config to %s? ", outPath)
	if !w.confirm("", true) {
		fmt.Println("Aborted — nothing written.")
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
	fmt.Printf("[✓] Config written to %s\n", outPath)

	// Validate.
	if _, err := config.Load(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "\n[!] Validation warning: %v\n", err)
		fmt.Fprintln(os.Stderr, "    Edit the file and run 'kongtrol config validate' when ready.")
	} else {
		fmt.Println("[✓] Config is valid.")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  kongtrol status                  — check tunnel states")
	fmt.Println("  kongtrol up <profile>            — connect a profile")
	fmt.Println("  kongtrol dashboard               — open the web UI")
	return nil
}

// ── profile collection ────────────────────────────────────────────────────────

func (w *wizard) collectProfile() (string, *yaml.Node, error) {
	name := w.prompt("  Profile name (e.g. office, aws, wg-home)", "")
	if name == "" {
		return "", nil, fmt.Errorf("profile name cannot be empty")
	}
	name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	fmt.Println("  Adapter types: forticlient | openvpn | protonvpn | ciscoanyconnect |")
	fmt.Println("                 wireguard | globalprotect | tailscale | cloudflarewarp")
	adapterType := w.prompt("  Type", "openvpn")

	fields := [][2]string{{"type", adapterType}}
	auth := config.AuthConfig{}

	switch adapterType {
	case "forticlient":
		fields = append(fields,
			[2]string{"host", w.prompt("  VPN host (e.g. vpn.empresa.com)", "")},
			[2]string{"port", w.promptDefault("  Port", "443")},
			[2]string{"tunnel_name", w.prompt("  Tunnel name (as shown in FortiClient UI)", "Office")},
		)
		ver := w.promptDefault("  FortiClient major version", "6")
		fields = append(fields, [2]string{"version", ver})
		auth.Method = w.promptDefault("  Auth method (certificate | credentials | certificate+credentials)", "certificate+credentials")
		if strings.Contains(auth.Method, "certificate") {
			auth.Cert = w.prompt("  Client cert path (e.g. ~/.kongtrol/certs/office.crt)", "")
			auth.Key = w.prompt("  Private key path (e.g. ~/.kongtrol/certs/office.key)", "")
		}
		if strings.Contains(auth.Method, "credentials") {
			auth.Username = w.prompt("  Username", "")
			auth.PasswordKeychain = name + ".password"
		}

	case "openvpn":
		fields = append(fields,
			[2]string{"config", w.prompt("  .ovpn config path (e.g. ~/.kongtrol/configs/server.ovpn)", "")},
		)
		auth.Method = w.promptDefault("  Auth method (certificate | credentials | certificate+credentials)", "certificate")
		if strings.Contains(auth.Method, "certificate") {
			cert := w.prompt("  Client cert path (leave blank if embedded in .ovpn)", "")
			key := w.prompt("  Private key path (leave blank if embedded in .ovpn)", "")
			auth.Cert = cert
			auth.Key = key
		}
		if strings.Contains(auth.Method, "credentials") {
			auth.Username = w.prompt("  Username", "")
			auth.PasswordKeychain = name + ".password"
		}

	case "protonvpn":
		fields = append(fields,
			[2]string{"server", w.promptDefault("  Server / country code (e.g. US, NL, fastest)", "fastest")},
			[2]string{"protocol", w.promptDefault("  Protocol (wireguard | openvpn)", "wireguard")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt("  ProtonVPN username", "")
		auth.PasswordKeychain = name + ".password"

	case "ciscoanyconnect":
		fields = append(fields,
			[2]string{"host", w.prompt("  VPN gateway host", "")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt("  Username", "")
		auth.PasswordKeychain = name + ".password"

	case "wireguard":
		fields = append(fields,
			[2]string{"config", w.prompt("  WireGuard .conf path (e.g. ~/.kongtrol/configs/wg0.conf)", "")},
		)
		auth.Method = "certificate" // keys are embedded in the .conf

	case "globalprotect":
		fields = append(fields,
			[2]string{"host", w.prompt("  GlobalProtect gateway host", "")},
		)
		auth.Method = "credentials"
		auth.Username = w.prompt("  Username", "")
		auth.PasswordKeychain = name + ".password"

	case "tailscale":
		exitNode := w.prompt("  Exit node hostname (leave blank to use Tailscale mesh routing)", "")
		if exitNode != "" {
			fields = append(fields, [2]string{"host", exitNode})
		}
		auth.Method = "credentials"
		useKey := w.confirm("  Use an auth key? (leave blank to reuse existing 'tailscale login' session)", false)
		if useKey {
			auth.PasswordKeychain = name + ".authkey"
		}

	case "cloudflarewarp":
		auth.Method = "credentials"
		fmt.Println("  [i] WARP uses no per-profile credentials.")
		fmt.Println("      Run 'warp-cli register' once if not already registered.")

	default:
		return "", nil, fmt.Errorf("unknown adapter type %q", adapterType)
	}

	// Priority.
	priorityStr := w.promptDefault("  Priority (lower = preferred, 1–100)", "10")
	fields = append(fields, [2]string{"priority", priorityStr})

	// Build auth sub-node.
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

	// Store password in keychain now if needed.
	if auth.PasswordKeychain != "" {
		if err := w.collectCredentials(name, adapterType, auth); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: could not store credential: %v\n", err)
		}
	}

	return name, node, nil
}

// collectCredentials prompts for sensitive values and stores them in the keychain.
func (w *wizard) collectCredentials(profileName, adapterType string, auth config.AuthConfig) error {
	switch adapterType {
	case "cloudflarewarp", "wireguard":
		// No runtime secrets needed.
		return nil
	case "tailscale":
		if auth.PasswordKeychain == "" {
			return nil
		}
		key := w.promptSecret("  Tailscale auth key (leave blank to skip)")
		if key != "" {
			return config.SetCredential(profileName, "password", key)
		}
		return nil
	default:
		if auth.PasswordKeychain != "" {
			pwd := w.promptSecret(fmt.Sprintf("  Password for %s (stored in OS keychain, not in YAML)", profileName))
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

func detectInstalledVPNs() []detectedVPN {
	type probe struct {
		label    string
		binaries []string
		args     []string
	}
	probes := []probe{
		{"FortiClient", []string{"fortivpn", "forticlientsslvpn"}, []string{"--version"}},
		{"OpenVPN", []string{"openvpn"}, []string{"--version"}},
		{"ProtonVPN", []string{"protonvpn-cli"}, []string{"--version"}},
		{"Cisco AnyConnect", []string{"vpn", "/opt/cisco/anyconnect/bin/vpn"}, []string{"-v"}},
		{"WireGuard", []string{"wg", "wg-quick"}, []string{"--version"}},
		{"GlobalProtect", []string{"globalprotect", "/opt/paloaltonetworks/globalprotect/pangpcrypt"}, []string{"--version"}},
		{"Tailscale", []string{"tailscale"}, []string{"version"}},
		{"Cloudflare WARP", []string{"warp-cli"}, []string{"--version"}},
	}

	var found []detectedVPN
	for _, p := range probes {
		for _, bin := range p.binaries {
			path, err := exec.LookPath(bin)
			if err != nil {
				// Also check common absolute paths.
				if _, statErr := os.Stat(bin); statErr != nil {
					continue
				}
				path = bin
			}
			ver := "installed"
			if out, err := exec.Command(path, p.args...).Output(); err == nil {
				ver = strings.TrimSpace(strings.Split(string(out), "\n")[0])
				if len(ver) > 60 {
					ver = ver[:60] + "…"
				}
			}
			found = append(found, detectedVPN{label: p.label, version: ver})
			break
		}
	}
	return found
}

// ── prompt helpers ────────────────────────────────────────────────────────────

func (w *wizard) banner() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║        Kongtrol — Setup Wizard           ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This wizard creates or updates ~/.kongtrol/kongtrol.yaml.")
	fmt.Println("Passwords are stored in your OS keychain — never in the YAML file.")
}

func (w *wizard) prompt(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func (w *wizard) promptDefault(label, def string) string {
	return w.prompt(label, def)
}

// promptSecret reads a line without echoing (best-effort; falls back to plain
// ReadString on platforms where terminal raw mode is unavailable).
func (w *wizard) promptSecret(label string) string {
	fmt.Printf("%s: ", label)
	// Try to suppress echo via stty (Unix). On Windows this silently fails and
	// we fall back to visible input — still stored in keychain, not in YAML.
	sttyOff := exec.Command("stty", "-echo")
	sttyOff.Stdin = os.Stdin
	_ = sttyOff.Run()

	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(line)

	sttyOn := exec.Command("stty", "echo")
	sttyOn.Stdin = os.Stdin
	_ = sttyOn.Run()
	fmt.Println() // newline after hidden input
	return line
}

func (w *wizard) confirm(label string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	if label != "" {
		fmt.Printf("%s [%s]: ", label, hint)
	} else {
		fmt.Printf("[%s]: ", hint)
	}
	line, _ := w.r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

// ── YAML document helpers ─────────────────────────────────────────────────────

// loadExisting attempts to read an existing config file and return both the
// parsed Config (for display) and the raw yaml.Node (for non-destructive editing).
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

	// Also parse into typed struct for display.
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, doc
	}
	return &cfg, doc
}

// freshDoc builds a minimal YAML mapping node with sensible defaults.
func (w *wizard) freshDoc() *yaml.Node {
	return mapNode([][2]string{
		// vpns mapping will be injected by collectProfile
	})
}

// scalarNode returns a plain YAML scalar node.
func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"}
}

// boolNode returns a boolean YAML scalar node.
func boolNode(val bool) *yaml.Node {
	v := "false"
	if val {
		v = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!bool"}
}

// intNode returns an integer YAML scalar node (from a string like "443").
func intNode(val string) *yaml.Node {
	if _, err := strconv.Atoi(val); err != nil {
		return scalarNode(val)
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!int"}
}

// mapNode builds a yaml.MappingNode from a list of [key, value] string pairs.
// Values that look like booleans use !!bool; integers use !!int; else !!str.
func mapNode(pairs [][2]string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range pairs {
		n.Content = append(n.Content, scalarNode(p[0]), autoScalar(p[1]))
	}
	return n
}

// autoScalar picks the right YAML tag based on the value.
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

// mappingKey returns the value node for a given key inside a mapping node,
// creating an empty mapping if the key does not exist.
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

// setMapping sets key → node inside a parent mapping, replacing if exists.
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
