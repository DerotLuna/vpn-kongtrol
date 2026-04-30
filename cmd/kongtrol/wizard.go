package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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

// ── Kong huh theme ────────────────────────────────────────────────────────────

func kongTheme() *huh.Theme {
	t := huh.ThemeBase()

	amber  := lipgloss.Color("220") // gold
	cyan   := lipgloss.Color("51")  // cyan / eye color
	dim    := lipgloss.Color("245") // gray
	bright := lipgloss.Color("255") // near-white
	green  := lipgloss.Color("82")  // success
	red    := lipgloss.Color("196") // error

	// Focused field styles (amber border + title)
	t.Focused.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(amber).
		Padding(0, 1)
	t.Focused.Title          = lipgloss.NewStyle().Foreground(amber).Bold(true)
	t.Focused.Description    = lipgloss.NewStyle().Foreground(dim)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(red)
	t.Focused.ErrorMessage   = lipgloss.NewStyle().Foreground(red)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	t.Focused.Option         = lipgloss.NewStyle().Foreground(bright)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(amber).Bold(true)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(green).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(dim)
	t.Focused.FocusedButton  = lipgloss.NewStyle().Foreground(bright).Background(amber).Bold(true).Padding(0, 2)
	t.Focused.BlurredButton  = lipgloss.NewStyle().Foreground(dim).Padding(0, 2)
	t.Focused.TextInput.Prompt      = lipgloss.NewStyle().Foreground(cyan)
	t.Focused.TextInput.Text        = lipgloss.NewStyle().Foreground(bright)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(dim)
	t.Focused.TextInput.Cursor      = lipgloss.NewStyle().Foreground(amber)
	t.Focused.NoteTitle = lipgloss.NewStyle().Foreground(amber).Bold(true)

	// Blurred (not focused)
	t.Blurred.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1)
	t.Blurred.Title          = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.Description    = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.Option         = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.SelectSelector = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(dim)

	// Group header
	t.Group.Title = lipgloss.NewStyle().Foreground(amber).Bold(true).Padding(0, 1)

	return t
}

// newForm wraps huh.NewForm with the Kong theme pre-applied.
func newForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(kongTheme())
}

// ── wizard entry point ────────────────────────────────────────────────────────

func runWizard(_ *cobra.Command, _ []string) error {
	// ── 1. Language selection ─────────────────────────────────────────────────
	var langChoice string
	langForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Language / Idioma").
				Options(
					huh.NewOption("Español", "es"),
					huh.NewOption("English", "en"),
				).
				Value(&langChoice),
		),
	).WithTheme(kongTheme())

	if err := langForm.Run(); err != nil {
		return nil // user aborted
	}
	var lang i18n.Lang
	if langChoice == "en" {
		lang = i18n.EN
	} else {
		lang = i18n.ES
	}
	t := func(key string) string              { return i18n.T(lang, key) }
	tf := func(key string, a ...any) string   { return i18n.F(lang, key, a...) }

	// ── 2. Logo + greeting ────────────────────────────────────────────────────
	fmt.Println()
	AnimateLogo(t("banner.subtitle"), version)
	fmt.Println()
	fmt.Println(styleDim.Render("  " + t("banner.yaml")))
	fmt.Println(styleDim.Render("  " + t("banner.keychain")))
	fmt.Println()

	// ── 3. Detect installed VPN clients ───────────────────────────────────────
	spin := newSpinner(t("detected.scanning"))
	spin.Start()
	detected := detectInstalledVPNs()
	spin.Stop()

	if len(detected) > 0 {
		fmt.Println(tuiInfo(styleBright.Render(t("detected.header"))))
		for _, d := range detected {
			fmt.Printf("    %s  %-22s  %s\n",
				styleOK.Render("✓"),
				styleBright.Render(d.label),
				styleDim.Render(d.version))
		}
	} else {
		fmt.Println(tuiWarn(t("detected.none")))
	}
	fmt.Println()

	// ── 4. Output path ────────────────────────────────────────────────────────
	home, _ := os.UserHomeDir()
	outPath := filepath.Join(home, ".kongtrol", "kongtrol.yaml")
	if cfgPath != "" {
		outPath = cfgPath
	}

	// ── 5. Load existing config ───────────────────────────────────────────────
	existing, existingRaw := loadExistingConfig(outPath)
	doc := existingRaw
	if doc == nil {
		doc = freshDoc()
	}

	// ── 6. Show existing profiles ─────────────────────────────────────────────
	if existing != nil && len(existing.VPNs) > 0 {
		SectionHeader(tf("existing.header", outPath, len(existing.VPNs)))
		for name, v := range existing.VPNs {
			fmt.Printf("    %s  %-16s  type=%s  host=%s\n",
				styleInfo.Render("·"),
				styleBright.Render(name),
				styleWarn.Render(v.Type),
				styleDim.Render(v.Host))
		}
		fmt.Println()

		// Offer credential refresh
		var refreshAny bool
		_ = newForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(t("profile.refresh_any")).
					Value(&refreshAny),
			),
		).Run()
		if refreshAny {
			for name, vpnCfg := range existing.VPNs {
				var refreshThis bool
				_ = newForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(tf("profile.refresh_creds") + " [" + name + "]").
							Value(&refreshThis),
					),
				).Run()
				if refreshThis {
					if err := collectCredentialsHuh(lang, name, vpnCfg.Type, vpnCfg.Auth); err != nil {
						fmt.Println(tuiWarn(err.Error()))
					}
				}
			}
		}
	}

	// ── 7. Add VPN profiles ───────────────────────────────────────────────────
	knownProfiles := make(map[string]bool)
	if existing != nil {
		for name := range existing.VPNs {
			knownProfiles[name] = true
		}
	}

	for {
		var addNew bool
		if err := newForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(styleBright.Render(t("profile.add_new"))).
					Value(&addNew),
			),
		).Run(); err != nil || !addNew {
			break
		}

		profile, vpnNode, err := collectProfileHuh(lang, detected, knownProfiles)
		if err != nil {
			fmt.Println(tuiErr(err.Error()))
			continue
		}
		if profile == "" {
			continue
		}

		vpnsNode := mappingKey(doc, "vpns")
		if vpnsNode == nil {
			vpnsNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			doc.Content = append(doc.Content, scalarNode("vpns"), vpnsNode)
		}

		if knownProfiles[profile] {
			fmt.Println(tuiWarn(tf("profile.already_exists", profile)))
			var replace bool
			_ = newForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(t("profile.replace_confirm")).
						Value(&replace),
				),
			).Run()
			if !replace {
				fmt.Println(styleDim.Render("  " + t("profile.replace_skipped")))
				continue
			}
			removeMapping(vpnsNode, profile)
		}

		vpnsNode.Content = append(vpnsNode.Content, scalarNode(profile), vpnNode)
		knownProfiles[profile] = true
	}

	// ── 8. Routing policies ───────────────────────────────────────────────────
	collectPoliciesHuh(lang, doc, existing, knownProfiles)

	// ── 9. Security ───────────────────────────────────────────────────────────
	SectionHeader(t("section.security"))

	secNode := mappingKey(doc, "security")
	if secNode == nil {
		secNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Content = append(doc.Content, scalarNode("security"), secNode)
	}

	var (
		enableKS      bool
		enableDNS     bool
		enableAudit   bool
		enableMonitor bool
	)
	// Pre-fill defaults from existing config.
	if existing != nil {
		enableKS      = existing.Security.KillSwitch.Enabled
		enableDNS     = existing.Security.DNSGuard.Enabled
		enableAudit   = existing.Security.AuditLog.Sign
		enableMonitor = existing.Monitor.Enabled
	} else {
		enableKS, enableDNS, enableAudit, enableMonitor = true, true, true, true
	}

	_ = newForm(
		huh.NewGroup(
			huh.NewNote().
				Title(t("section.security")).
				Description(styleDim.Render("Configure security policies for all tunnels.")),
			huh.NewConfirm().
				Title(t("security.kill_switch")).
				Description(styleDim.Render(t("hint.killswitch"))).
				Value(&enableKS),
			huh.NewConfirm().
				Title(t("security.dns_guard")).
				Description(styleDim.Render(t("hint.dnsguard"))).
				Value(&enableDNS),
			huh.NewConfirm().
				Title(t("security.audit_log")).
				Description(styleDim.Render(t("hint.auditlog"))).
				Value(&enableAudit),
			huh.NewConfirm().
				Title(t("monitor.dashboard")).
				Description(styleDim.Render(t("hint.dashboard"))).
				Value(&enableMonitor),
		),
	).Run()

	auditPath := filepath.Join(home, ".kongtrol", "audit.log")
	if enableKS {
		setMapping(secNode, "kill_switch", mapNode([][2]string{
			{"enabled", "true"}, {"mode", "strict"}, {"allow_lan", "true"},
		}))
	}
	if enableDNS {
		setMapping(secNode, "dns_guard", mapNode([][2]string{
			{"enabled", "true"}, {"fallback_dns", "1.1.1.1"},
		}))
	}
	if enableAudit {
		setMapping(secNode, "audit_log", mapNode([][2]string{
			{"path", auditPath}, {"max_size_mb", "100"}, {"sign", "true"},
		}))
	}
	if enableMonitor {
		setMapping(doc, "monitor", mapNode([][2]string{{"enabled", "true"}}))
	}

	// ── 10. Confirm + write ───────────────────────────────────────────────────
	fmt.Println()
	var doWrite bool
	_ = newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(tf("write.confirm", styleWarn.Render(outPath))).
				Value(&doWrite),
		),
	).Run()

	if !doWrite {
		fmt.Println(tuiWarn(t("write.aborted")))
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

	fmt.Println(tuiOK(styleBright.Render(tf("write.success", outPath))))

	if _, err := config.Load(outPath); err != nil {
		fmt.Fprintln(os.Stderr, tuiWarn(tf("write.validation_warn", err)))
		fmt.Fprintln(os.Stderr, styleDim.Render(t("write.validation_hint")))
	} else {
		fmt.Println(tuiOK(styleOK.Render(t("write.valid"))))
	}

	// ── Next steps ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(styleGold.Render(t("nextsteps.header")))
	fmt.Println("  " + stylePrompt.Render(t("nextsteps.init")))
	fmt.Println("  " + stylePrompt.Render(t("nextsteps.status")))
	fmt.Println("  " + stylePrompt.Render(t("nextsteps.up")))
	fmt.Println("  " + stylePrompt.Render(t("nextsteps.dashboard")))
	fmt.Println()
	return nil
}

// ── Profile collection ────────────────────────────────────────────────────────

func collectProfileHuh(lang i18n.Lang, detected []detectedVPN, knownProfiles map[string]bool) (string, *yaml.Node, error) {
	t := func(key string) string { return i18n.T(lang, key) }

	SectionHeader(t("section.new_profile"))

	// Build adapter options — detected ones first, with a note.
	allAdapters := []string{"forticlient", "openvpn", "protonvpn", "ciscoanyconnect", "wireguard", "globalprotect", "tailscale", "cloudflarewarp"}
	detectedKeys := detectedAdapterKeys(detected)

	var adapterOpts []huh.Option[string]
	for _, key := range allAdapters {
		if detectedKeys[key] {
			adapterOpts = append(adapterOpts, huh.NewOption(key+" ✓", key))
		}
	}
	for _, key := range allAdapters {
		if !detectedKeys[key] {
			adapterOpts = append(adapterOpts, huh.NewOption(key, key))
		}
	}

	var (
		name        string
		adapterType string
	)
	defAdapter := "openvpn"
	if len(adapterOpts) > 0 {
		// First option is the first detected (or first in list).
		defAdapter = allAdapters[0]
		for _, key := range allAdapters {
			if detectedKeys[key] {
				defAdapter = key
				break
			}
		}
	}
	adapterType = defAdapter

	if err := newForm(
		huh.NewGroup(
			huh.NewInput().
				Title(t("collect.profile_name")).
				Description(styleDim.Render("e.g. work-vpn, office-forti")).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("%s", t("collect.profile_name_empty"))
					}
					return nil
				}).
				Value(&name),
			huh.NewSelect[string]().
				Title(t("collect.type")).
				Description(styleDim.Render("✓ = detected on this machine")).
				Options(adapterOpts...).
				Value(&adapterType),
		),
	).Run(); err != nil {
		return "", nil, nil
	}

	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
	if name == "" {
		return "", nil, fmt.Errorf("%s", t("collect.profile_name_empty"))
	}

	// Type-specific fields
	fields := [][2]string{{"type", adapterType}}
	auth := config.AuthConfig{}

	switch adapterType {
	case "forticlient":
		if err := collectFortiFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "openvpn":
		if err := collectOpenVPNFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "protonvpn":
		if err := collectProtonFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "ciscoanyconnect":
		if err := collectCiscoFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "wireguard":
		if err := collectWireGuardFields(lang, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "globalprotect":
		if err := collectGlobalProtectFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "tailscale":
		if err := collectTailscaleFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "cloudflarewarp":
		fmt.Println(tuiInfo(i18n.T(lang, "collect.warp_info1")))
		fmt.Println(styleDim.Render("  " + i18n.T(lang, "collect.warp_info2")))
		auth.Method = "credentials"

	default:
		return "", nil, fmt.Errorf("%s", i18n.F(lang, "collect.unknown_adapter", adapterType))
	}

	// Binary path override when not detected
	if adapterType != "cloudflarewarp" && !detectedKeys[adapterType] {
		fmt.Println(tuiWarn(i18n.T(lang, "collect.not_detected")))
		var binPath string
		_ = newForm(huh.NewGroup(
			huh.NewInput().
				Title(i18n.T(lang, "collect.binary_path")).
				Description(styleDim.Render("Leave empty to skip")).
				Value(&binPath),
		)).Run()
		if binPath != "" {
			fields = append(fields, [2]string{"binary_path", binPath})
		}
	}

	// Priority
	var priorityStr string = "10"
	_ = newForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, "collect.priority")).
			Placeholder("10").
			Value(&priorityStr),
	)).Run()
	if priorityStr == "" {
		priorityStr = "10"
	}
	fields = append(fields, [2]string{"priority", priorityStr})

	// Build YAML node
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
		if err := collectCredentialsHuh(lang, name, adapterType, auth); err != nil {
			fmt.Println(tuiWarn(i18n.F(lang, "collect.password_warn", err)))
		}
	}

	return name, node, nil
}

// ── Type-specific field collectors ───────────────────────────────────────────

func collectFortiFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		host       string
		port       string = "443"
		tunnelName string = "Office"
		version    string = "6"
	)

	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.host")).
			Description(styleDim.Render(i18n.T(lang, "hint.host.forti"))).
			Value(&host),
		huh.NewInput().Title(i18n.T(lang, "collect.port")).
			Placeholder("443").Value(&port),
		huh.NewInput().Title(i18n.T(lang, "collect.tunnel_name")).
			Placeholder("Office").Value(&tunnelName),
		huh.NewSelect[string]().Title(i18n.T(lang, "collect.forti_ver")).
			Options(
				huh.NewOption("6.x ("+i18n.T(lang, "forti.ver.hint.6")+")", "6"),
				huh.NewOption("7.x ("+i18n.T(lang, "forti.ver.hint.7")+")", "7"),
				huh.NewOption("5.x ("+i18n.T(lang, "forti.ver.hint.5")+")", "5"),
			).Value(&version),
	)).Run(); err != nil {
		return nil
	}
	if port == "" {
		port = "443"
	}
	if tunnelName == "" {
		tunnelName = "Office"
	}

	*fields = append(*fields,
		[2]string{"host", host},
		[2]string{"port", port},
		[2]string{"tunnel_name", tunnelName},
		[2]string{"version", version},
	)

	if runtime.GOOS == "windows" {
		fmt.Println(styleDim.Render(i18n.T(lang, "hint.auth.forti.win")))
		auth.Method = "credentials"
		auth.Username = promptInput(lang, "collect.username", "")
		auth.PasswordKeychain = name + ".password"
	} else {
		auth.Method = selectAuthMethod(lang, "certificate+credentials")
		if strings.Contains(auth.Method, "certificate") {
			auth.Cert = promptInput(lang, "collect.cert", "")
			auth.Key = promptInput(lang, "collect.key", "")
		}
		if strings.Contains(auth.Method, "credentials") {
			auth.Username = promptInput(lang, "collect.username", "")
			auth.PasswordKeychain = name + ".password"
		}
	}
	return nil
}

func collectOpenVPNFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var ovpnConfig string
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.ovpn_config")).
			Description(styleDim.Render(i18n.T(lang, "hint.ovpn_config"))).
			Value(&ovpnConfig),
	)).Run(); err != nil {
		return nil
	}
	*fields = append(*fields, [2]string{"config", ovpnConfig})
	auth.Method = selectAuthMethod(lang, "certificate")
	if strings.Contains(auth.Method, "certificate") {
		auth.Cert = promptInput(lang, "collect.ovpn_cert", "")
		auth.Key = promptInput(lang, "collect.ovpn_key", "")
	}
	if strings.Contains(auth.Method, "credentials") {
		auth.Username = promptInput(lang, "collect.username", "")
		auth.PasswordKeychain = name + ".password"
	}
	return nil
}

func collectProtonFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		server   string = "fastest"
		protocol string = "wireguard"
	)
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.proton_srv")).
			Description(styleDim.Render(i18n.T(lang, "hint.proton_srv"))).
			Placeholder("fastest").Value(&server),
		huh.NewSelect[string]().Title(i18n.T(lang, "collect.proton_proto")).
			Options(
				huh.NewOption("WireGuard — "+i18n.T(lang, "proto.hint.wireguard"), "wireguard"),
				huh.NewOption("OpenVPN — "+i18n.T(lang, "proto.hint.openvpn"), "openvpn"),
			).Value(&protocol),
	)).Run(); err != nil {
		return nil
	}
	if server == "" {
		server = "fastest"
	}
	*fields = append(*fields, [2]string{"server", server}, [2]string{"protocol", protocol})
	auth.Method = "credentials"
	auth.Username = promptInput(lang, "collect.proton_user", "")
	auth.PasswordKeychain = name + ".password"
	return nil
}

func collectCiscoFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var host string
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.cisco_host")).
			Description(styleDim.Render(i18n.T(lang, "hint.host.cisco"))).
			Value(&host),
	)).Run(); err != nil {
		return nil
	}
	*fields = append(*fields, [2]string{"host", host})
	auth.Method = "credentials"
	auth.Username = promptInput(lang, "collect.cisco_user", "")
	auth.PasswordKeychain = name + ".password"
	return nil
}

func collectWireGuardFields(lang i18n.Lang, fields *[][2]string, auth *config.AuthConfig) error {
	var wgConfig string
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.wg_config")).
			Description(styleDim.Render(i18n.T(lang, "hint.wg_config"))).
			Value(&wgConfig),
	)).Run(); err != nil {
		return nil
	}
	*fields = append(*fields, [2]string{"config", wgConfig})
	auth.Method = "certificate"
	return nil
}

func collectGlobalProtectFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var host string
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.gp_host")).
			Description(styleDim.Render(i18n.T(lang, "hint.host.gp"))).
			Value(&host),
	)).Run(); err != nil {
		return nil
	}
	*fields = append(*fields, [2]string{"host", host})
	auth.Method = "credentials"
	auth.Username = promptInput(lang, "collect.gp_user", "")
	auth.PasswordKeychain = name + ".password"
	return nil
}

func collectTailscaleFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		exitNode string
		useKey   bool
	)
	if err := newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.ts_exitnode")).
			Description(styleDim.Render("Leave empty for default routing")).
			Value(&exitNode),
		huh.NewConfirm().Title(i18n.T(lang, "collect.ts_usekey")).
			Value(&useKey),
	)).Run(); err != nil {
		return nil
	}
	if exitNode != "" {
		*fields = append(*fields, [2]string{"host", exitNode})
	}
	auth.Method = "credentials"
	if useKey {
		auth.PasswordKeychain = name + ".authkey"
	}
	return nil
}

// selectAuthMethod shows an auth method selector and returns the chosen value.
func selectAuthMethod(lang i18n.Lang, def string) string {
	var method string = def
	_ = newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(i18n.T(lang, "collect.auth_method")).
			Options(
				huh.NewOption(i18n.T(lang, "auth.hint.credentials"), "credentials"),
				huh.NewOption(i18n.T(lang, "auth.hint.certificate"), "certificate"),
				huh.NewOption(i18n.T(lang, "auth.hint.cert+creds"), "certificate+credentials"),
			).
			Value(&method),
	)).Run()
	return method
}

// promptInput shows a single-field huh form and returns the value.
func promptInput(lang i18n.Lang, key, placeholder string) string {
	var val string
	_ = newForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, key)).
			Placeholder(placeholder).
			Value(&val),
	)).Run()
	return strings.TrimSpace(val)
}

// ── Credentials ───────────────────────────────────────────────────────────────

func collectCredentialsHuh(lang i18n.Lang, profileName, adapterType string, auth config.AuthConfig) error {
	switch adapterType {
	case "cloudflarewarp", "wireguard":
		return nil
	case "tailscale":
		if auth.PasswordKeychain == "" {
			return nil
		}
		var key string
		if err := newForm(huh.NewGroup(
			huh.NewInput().
				Title(i18n.T(lang, "collect.ts_key")).
				EchoMode(huh.EchoModePassword).
				Value(&key),
		)).Run(); err != nil {
			return nil
		}
		if key != "" {
			return config.SetCredential(profileName, "password", key)
		}
		return nil
	default:
		if auth.PasswordKeychain == "" {
			return nil
		}
		var pwd string
		title := i18n.F(lang, "collect.password", styleBright.Render(profileName))
		if err := newForm(huh.NewGroup(
			huh.NewInput().
				Title(title).
				EchoMode(huh.EchoModePassword).
				Value(&pwd),
		)).Run(); err != nil {
			return nil
		}
		if pwd != "" {
			return config.SetCredential(profileName, "password", pwd)
		}
		return nil
	}
}

// ── Routing policies ──────────────────────────────────────────────────────────

func collectPoliciesHuh(lang i18n.Lang, doc *yaml.Node, existing *config.Config, profileNames map[string]bool) {
	SectionHeader(i18n.T(lang, "section.policies"))

	if existing != nil && len(existing.Policies) > 0 {
		fmt.Println(tuiInfo(i18n.T(lang, "policy.existing")))
		for _, p := range existing.Policies {
			fmt.Printf("    %s  %s → %s\n",
				styleInfo.Render("·"),
				styleBright.Render(p.Name),
				styleWarn.Render(p.Via))
		}
		fmt.Println()
	}

	// Build profile options for "via" selector.
	var profileList []string
	for name := range profileNames {
		profileList = append(profileList, name)
	}
	sort.Strings(profileList)

	if len(profileList) == 0 {
		fmt.Println(styleDim.Render("    " + i18n.T(lang, "policy.no_profiles")))
		return
	}

	viaOpts := make([]huh.Option[string], len(profileList))
	for i, n := range profileList {
		viaOpts[i] = huh.NewOption(n, n)
	}

	policiesNode := mappingKey(doc, "policies")
	if policiesNode == nil {
		policiesNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		doc.Content = append(doc.Content, scalarNode("policies"), policiesNode)
	}

	knownPolicies := make(map[string]bool)
	if existing != nil {
		for _, p := range existing.Policies {
			knownPolicies[p.Name] = true
		}
	}

	added := 0
	for {
		defAdd := added == 0 && len(profileList) > 0
		var addNew bool
		if defAdd {
			addNew = true
		}
		if err := newForm(huh.NewGroup(
			huh.NewConfirm().
				Title(i18n.T(lang, "policy.add_new")).
				Value(&addNew),
		)).Run(); err != nil || !addNew {
			break
		}

		var (
			policyName string
			via        string = profileList[0]
			domainsRaw string
			ipsRaw     string
		)

		if err := newForm(
			huh.NewGroup(
				huh.NewInput().
					Title(i18n.T(lang, "policy.name")).
					Description(styleDim.Render("e.g. work-internal, saas-apps")).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("policy name required")
						}
						return nil
					}).
					Value(&policyName),
				huh.NewSelect[string]().
					Title(i18n.T(lang, "policy.via")).
					Options(viaOpts...).
					Value(&via),
			),
			huh.NewGroup(
				huh.NewNote().
					Title(i18n.T(lang, "policy.domains_hint")).
					Description(styleDim.Render("e.g. internal.company.com, *.corp")),
				huh.NewInput().
					Title(i18n.T(lang, "policy.domain_prompt")).
					Description(styleDim.Render("Comma-separated, leave empty to skip")).
					Value(&domainsRaw),
				huh.NewNote().
					Title(i18n.T(lang, "policy.ips_hint")).
					Description(styleDim.Render("e.g. 10.0.0.0/8, 192.168.1.0/24")),
				huh.NewInput().
					Title(i18n.T(lang, "policy.ip_prompt")).
					Description(styleDim.Render("Comma-separated CIDR, leave empty to skip")).
					Value(&ipsRaw),
			),
		).Run(); err != nil {
			break
		}

		policyName = strings.TrimSpace(policyName)
		if policyName == "" {
			continue
		}

		var domains []string
		for _, d := range strings.Split(domainsRaw, ",") {
			if v := strings.TrimSpace(d); v != "" {
				domains = append(domains, v)
			}
		}
		var ipRanges []string
		for _, ip := range strings.Split(ipsRaw, ",") {
			if v := strings.TrimSpace(ip); v != "" {
				ipRanges = append(ipRanges, v)
			}
		}

		if len(domains) == 0 && len(ipRanges) == 0 {
			fmt.Println(tuiWarn(i18n.T(lang, "policy.empty_match")))
			continue
		}

		if knownPolicies[policyName] {
			fmt.Println(tuiWarn(i18n.F(lang, "policy.already_exists", policyName)))
			var replace bool
			_ = newForm(huh.NewGroup(
				huh.NewConfirm().Title(i18n.T(lang, "policy.replace_confirm")).Value(&replace),
			)).Run()
			if !replace {
				fmt.Println(styleDim.Render(i18n.T(lang, "policy.replace_skipped")))
				continue
			}
			removePolicyByName(policiesNode, policyName)
		}

		policiesNode.Content = append(policiesNode.Content, policyNode(policyName, via, domains, ipRanges))
		knownPolicies[policyName] = true
		added++
	}

	fmt.Println()
	fmt.Println(styleDim.Render("    " + i18n.T(lang, "policy.yaml_hint")))
}

// policyNode builds a YAML mapping node for a single policy rule.
func policyNode(name, via string, domains, ipRanges []string) *yaml.Node {
	matchNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(domains) > 0 {
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, d := range domains {
			seq.Content = append(seq.Content, scalarNode(d))
		}
		matchNode.Content = append(matchNode.Content, scalarNode("domains"), seq)
	}
	if len(ipRanges) > 0 {
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, ip := range ipRanges {
			seq.Content = append(seq.Content, scalarNode(ip))
		}
		matchNode.Content = append(matchNode.Content, scalarNode("ip_ranges"), seq)
	}
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	n.Content = append(n.Content,
		scalarNode("name"), scalarNode(name),
		scalarNode("match"), matchNode,
		scalarNode("via"), scalarNode(via),
	)
	return n
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
	for _, line := range strings.Split(string(out), "\n") {
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

// ── YAML document helpers ─────────────────────────────────────────────────────

func loadExistingConfig(path string) (*config.Config, *yaml.Node) {
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

func freshDoc() *yaml.Node {
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

func removePolicyByName(seq *yaml.Node, name string) {
	if seq == nil {
		return
	}
	for i, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(item.Content); j += 2 {
			if item.Content[j].Value == "name" && item.Content[j+1].Value == name {
				seq.Content = append(seq.Content[:i], seq.Content[i+1:]...)
				return
			}
		}
	}
}

func removeMapping(parent *yaml.Node, key string) {
	if parent == nil {
		return
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return
		}
	}
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
