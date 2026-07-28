package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	initCmd.Short = ct("cli.init.short")
	initCmd.Example = ct("cli.init.examples")
	rootCmd.AddCommand(initCmd)
}

// ── Kong huh theme ────────────────────────────────────────────────────────────

func kongTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Signal Contour tokens (theme.go)
	amber := colSignal
	cyan := colSteel
	dim := colDim
	bright := colText
	green := colSignal
	red := colDanger

	// Focused field styles (amber border + title)
	t.Focused.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(amber).
		Padding(0, 1)
	t.Focused.Title = lipgloss.NewStyle().Foreground(amber).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(dim)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(red)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(red)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	t.Focused.Option = lipgloss.NewStyle().Foreground(bright)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(amber).Bold(true)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(green).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(dim)
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(bright).Background(amber).Bold(true).Padding(0, 2)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(dim).Padding(0, 2)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(cyan)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(bright)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(dim)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(amber)
	t.Focused.NoteTitle = lipgloss.NewStyle().Foreground(amber).Bold(true)

	// Blurred (not focused)
	t.Blurred.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.HiddenBorder()).
		Padding(0, 1)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.Option = lipgloss.NewStyle().Foreground(dim)
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

// ── Cancellation ──────────────────────────────────────────────────────────────

// errWizardCancelled is returned up the call stack the moment a user aborts
// any huh form (Ctrl+C / Esc) — every collect* function propagates it instead
// of swallowing the error and continuing with zero-value fields.
var errWizardCancelled = errors.New("wizard cancelled")

// runForm runs a huh.Form and normalizes huh's own abort error to
// errWizardCancelled, so every call site can check for one sentinel.
func runForm(form *huh.Form) error {
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errWizardCancelled
		}
		return err
	}
	return nil
}

// ── wizard entry point ────────────────────────────────────────────────────────

func runWizard(_ *cobra.Command, _ []string) error {
	// ── 1. Language selection — only asked the very first time ───────────────
	// Once a language is on record in preferences.json, every later `init` run
	// (and every other command, via cliLang()) just uses it. Changing it after
	// that goes through `kongtrol config lang <es|en>`, not this wizard.
	prefs, prefsErr := loadPreferences()
	var langChoice string
	firstRun := prefsErr != nil || strings.TrimSpace(prefs.Language) == ""

	if firstRun {
		langForm := newForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Language / Idioma").
					Options(
						huh.NewOption("Español", "es"),
						huh.NewOption("English", "en"),
					).
					Value(&langChoice),
			),
		)
		if err := runForm(langForm); err != nil {
			return nil // user aborted before there's anything to lose
		}
	} else {
		langChoice = strings.ToLower(strings.TrimSpace(prefs.Language))
	}

	var lang i18n.Lang
	if langChoice == "en" {
		lang = i18n.EN
	} else {
		lang = i18n.ES
	}
	t := func(key string) string { return i18n.T(lang, key) }
	tf := func(key string, a ...any) string { return i18n.F(lang, key, a...) }
	cancelled := func() error {
		fmt.Println(styleWarn.Render(t("wizard.cancelled")))
		return nil
	}

	if firstRun {
		// Persist the choice so every other command (status, up, ...) uses it
		// too, and won't ask again.
		if p, err := loadPreferences(); err == nil {
			p.Language = langChoice
			_ = savePreferences(p)
		}
		fmt.Println(styleDim.Render("  " + t("wizard.lang.change_hint")))
	}

	// ── 2. Banner — the site hero's motd, now in the terminal ─────────────────
	fmt.Println()
	AnimateBanner(t("banner.subtitle"), version)
	fmt.Println(styleDim.Render("  " + sym("#", "#") + " " + t("banner.yaml")))
	fmt.Println(styleDim.Render("  " + sym("#", "#") + " " + t("banner.keychain")))
	fmt.Println()

	// ── 3. Detect installed VPN clients — boot-sequence reveal ────────────────
	spin := newSpinner(t("detected.scanning"))
	spin.Start()
	detected := detectInstalledVPNs()
	spin.Stop()

	if len(detected) > 0 {
		fmt.Println(tuiInfo(styleBright.Render(t("detected.header"))))
		for _, d := range detected {
			if isTerminal() {
				time.Sleep(90 * time.Millisecond)
			}
			fmt.Printf("    %s  %-22s  %s\n",
				styleOK.Render(sym("✓", "[OK]")),
				styleBright.Render(d.label),
				styleInfo.Render(d.version))
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
		if err := runForm(newForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(t("profile.refresh_any")).
					Value(&refreshAny),
			),
		)); err != nil {
			return cancelled()
		}
		if refreshAny {
			for name, vpnCfg := range existing.VPNs {
				var refreshThis bool
				if err := runForm(newForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(tf("profile.refresh_creds") + " [" + name + "]").
							Value(&refreshThis),
					),
				)); err != nil {
					return cancelled()
				}
				if refreshThis {
					if err := collectCredentialsHuh(lang, name, vpnCfg.Type, vpnCfg.Auth); err != nil {
						if errors.Is(err, errWizardCancelled) {
							return cancelled()
						}
						fmt.Println(tuiWarn(err.Error()))
					}
				}
			}
		}
	}

	// ── 7. Add VPN profiles ───────────────────────────────────────────────────
	StepHeader(1, 4, t("section.profiles"))

	knownProfiles := make(map[string]bool)
	if existing != nil {
		for name := range existing.VPNs {
			knownProfiles[name] = true
		}
	}

	var addedProfiles []profileSummary
	for {
		var addNew bool
		if err := runForm(newForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(styleBright.Render(t("profile.add_new"))).
					Value(&addNew),
			),
		)); err != nil || !addNew {
			break
		}

		profile, vpnNode, err := collectProfileHuh(lang, detected, knownProfiles)
		if err != nil {
			if errors.Is(err, errWizardCancelled) {
				return cancelled()
			}
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
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(t("profile.replace_confirm")).
						Value(&replace),
				),
			)); err != nil {
				return cancelled()
			}
			if !replace {
				fmt.Println(styleDim.Render("  " + t("profile.replace_skipped")))
				continue
			}
			removeMapping(vpnsNode, profile)
		}

		vpnsNode.Content = append(vpnsNode.Content, scalarNode(profile), vpnNode)
		knownProfiles[profile] = true
		addedProfiles = append(addedProfiles, profileSummary{
			name: profile,
			typ:  valueOf(vpnNode, "type"),
			host: valueOf(vpnNode, "host"),
		})
	}

	// ── 8. Routing policies ───────────────────────────────────────────────────
	addedPolicies, err := collectPoliciesHuh(lang, doc, existing, knownProfiles)
	if err != nil {
		if errors.Is(err, errWizardCancelled) {
			return cancelled()
		}
		return err
	}

	// ── 9. Security ───────────────────────────────────────────────────────────
	StepHeader(3, 4, t("section.security"))

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
		enableKS = existing.Security.KillSwitch.Enabled
		enableDNS = existing.Security.DNSGuard.Enabled
		enableAudit = existing.Security.AuditLog.Sign
		enableMonitor = existing.Monitor.Enabled
	} else {
		enableKS, enableDNS, enableAudit, enableMonitor = true, true, true, true
	}

	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewNote().
				Title(t("section.security")).
				Description(styleDim.Render(t("security.note"))),
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
	)); err != nil {
		return cancelled()
	}

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

	// ── 10. Review + confirm + write ──────────────────────────────────────────
	StepHeader(4, 4, t("section.write"))
	fmt.Println(renderReviewPanel(lang, outPath, addedProfiles, addedPolicies, securitySummary{
		killSwitch: enableKS, dnsGuard: enableDNS, auditLog: enableAudit, monitor: enableMonitor,
	}))
	fmt.Println()

	var doWrite bool
	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(tf("write.confirm", styleWarn.Render(outPath))).
				Value(&doWrite),
		),
	)); err != nil {
		return cancelled()
	}

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
	fmt.Println(NextStepsPanel(t("nextsteps.header"), []string{
		t("nextsteps.init"),
		t("nextsteps.status"),
		t("nextsteps.up"),
		t("nextsteps.dashboard"),
	}))
	fmt.Println()
	return nil
}

// ── Review panel ──────────────────────────────────────────────────────────────

// profileSummary is the minimal shape shown in the pre-write review panel.
type profileSummary struct {
	name, typ, host string
}

// securitySummary mirrors the four toggles collected in step 3.
type securitySummary struct {
	killSwitch, dnsGuard, auditLog, monitor bool
}

// valueOf reads a scalar field out of a profile's yaml.Node mapping — used to
// summarize a freshly-built node without re-parsing the document.
func valueOf(node *yaml.Node, key string) string {
	if v := mappingKey(node, key); v != nil {
		return v.Value
	}
	return ""
}

// renderReviewPanel renders a rounded-border summary of everything that is
// about to be written to disk, shown right before the final write confirm so
// the user isn't confirming blind.
func renderReviewPanel(lang i18n.Lang, outPath string, profiles []profileSummary, policies []string, sec securitySummary) string {
	var b strings.Builder
	b.WriteString(stylePrompt.Render(sym("◎", "#")) + "  " + styleGold.Render(i18n.T(lang, "review.header")))

	b.WriteString("\n\n" + styleBright.Render(i18n.T(lang, "review.profiles")) + ":")
	if len(profiles) == 0 {
		b.WriteString("\n  " + styleDim.Render(i18n.T(lang, "review.none")))
	} else {
		for _, p := range profiles {
			fmt.Fprintf(&b, "\n  · %s  type=%s  host=%s",
				styleBright.Render(p.name), styleWarn.Render(p.typ), styleDim.Render(p.host))
		}
	}

	b.WriteString("\n\n" + styleBright.Render(i18n.T(lang, "review.policies")) + ":")
	if len(policies) == 0 {
		b.WriteString("\n  " + styleDim.Render(i18n.T(lang, "review.none")))
	} else {
		for _, name := range policies {
			b.WriteString("\n  · " + styleBright.Render(name))
		}
	}

	b.WriteString("\n\n" + styleBright.Render(i18n.T(lang, "review.security")) + ":")
	fmt.Fprintf(&b, "\n  kill_switch=%s  dns_guard=%s  audit_log=%s  dashboard=%s",
		onOff(sec.killSwitch), onOff(sec.dnsGuard), onOff(sec.auditLog), onOff(sec.monitor))

	b.WriteString("\n\n" + styleDim.Render(outPath))

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colEmber2).
		Padding(0, 2).
		Render(b.String())
	return "  " + strings.ReplaceAll(box, "\n", "\n  ")
}

func onOff(v bool) string {
	if v {
		return styleOK.Render("on")
	}
	return styleDim.Render("off")
}
