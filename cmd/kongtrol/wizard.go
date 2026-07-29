package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

// wizardKeyMap rebinds huh's Quit signal from Ctrl+C to Esc: Esc aborts only
// the *current* form (surfaced as errWizardCancelled — every collect*
// function treats that as "back one level", e.g. returning to the edit
// menu). Ctrl+C is handled separately, outside of huh entirely (see
// formGuard below), as the one true "close everything" signal.
func wizardKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("esc"))
	return km
}

// wizardLang is the wizard's resolved display language, set once by
// selectWizardLanguage at the top of runWizard. formGuard reads it to render
// the nav-hint footer (below) in the right language without threading lang
// through every one of the wizard's ~15 runForm call sites individually.
var wizardLang i18n.Lang

// newForm wraps huh.NewForm with the Kong theme and wizardKeyMap pre-applied,
// and disables huh's own (English-only, per-field) help footer — formGuard
// renders one consistent, fully localized hint line instead, on every form.
func newForm(groups ...*huh.Group) *huh.Form {
	for _, g := range groups {
		g.WithShowHelp(false)
	}
	return huh.NewForm(groups...).WithTheme(kongTheme()).WithKeyMap(wizardKeyMap())
}

// ── Cancellation ──────────────────────────────────────────────────────────────

// errWizardCancelled is returned up the call stack the moment a user backs
// out of a single form with Esc — every collect* function propagates it
// instead of swallowing the error and continuing with zero-value fields.
var errWizardCancelled = errors.New("wizard cancelled")

// errWizardQuit signals a hard Ctrl+C: the user wants to close the wizard
// entirely, without saving, no matter how deep in a nested menu/action they
// are. runForm panics with it instead of returning it, and it is recovered
// exactly once at the top of runWizard — that lets it unwind every
// intermediate loop/switch (edit menu, add-profile retry loop, ...) without
// having to thread a second sentinel error through each of their call sites
// alongside errWizardCancelled.
var errWizardQuit = errors.New("wizard quit")

// formGuard wraps a *huh.Form as its own tea.Model so Ctrl+C can be
// intercepted before it ever reaches huh's own key handling (which, with
// wizardKeyMap, no longer binds Quit to Ctrl+C at all).
type formGuard struct {
	form     *huh.Form
	hardQuit bool
}

func (g *formGuard) Init() tea.Cmd { return g.form.Init() }

func (g *formGuard) View() string {
	return g.form.View() + "\n" + styleDim.Render(i18n.T(wizardLang, "hint.nav.list"))
}

func (g *formGuard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		g.hardQuit = true
		return g, tea.Quit
	}
	m, cmd := g.form.Update(msg)
	g.form = m.(*huh.Form)
	return g, cmd
}

// runForm runs a huh.Form through formGuard and normalizes the outcome:
// Esc → errWizardCancelled, Ctrl+C → panic(errWizardQuit) (see formGuard
// doc), normal completion → nil.
func runForm(form *huh.Form) error {
	// form.Run()/RunWithContext() normally set these before starting the
	// program — they're what tells bubbletea to actually quit on submit
	// (SubmitCmd) or on the Quit keybinding firing (CancelCmd). Bypassing
	// Run() to wrap the form in formGuard means we have to set them
	// ourselves, or the form completes internally (State flips to
	// StateCompleted/StateAborted) but the program never receives a signal
	// to exit — it just hangs on a blank screen forever.
	form.SubmitCmd = tea.Quit
	form.CancelCmd = tea.Interrupt

	guard := &formGuard{form: form}
	m, err := tea.NewProgram(guard).Run()
	if err != nil && !errors.Is(err, tea.ErrInterrupted) {
		return fmt.Errorf("huh: %w", err)
	}
	g := m.(*formGuard)
	if g.hardQuit {
		panic(errWizardQuit)
	}
	if g.form.State == huh.StateAborted {
		return errWizardCancelled
	}
	return nil
}

// ── wizard entry point ────────────────────────────────────────────────────────

func runWizard(_ *cobra.Command, _ []string) (err error) {
	// Recover the one and only panic runForm ever raises (errWizardQuit, on
	// Ctrl+C) — see its doc comment for why this unwinds via panic/recover
	// instead of a second sentinel threaded through every call site. lang
	// may not be set yet if Ctrl+C hits the very first (language) form, so
	// the message falls back to the zero value of i18n.Lang (Spanish).
	var lang i18n.Lang
	defer func() {
		if r := recover(); r != nil {
			quitErr, ok := r.(error)
			if !ok || !errors.Is(quitErr, errWizardQuit) {
				panic(r)
			}
			fmt.Println(styleWarn.Render(i18n.T(lang, "wizard.quit")))
			err = nil
		}
	}()

	// ── 1. Language selection — only asked the very first time ───────────────
	var firstRun bool
	lang, firstRun, err = selectWizardLanguage()
	if err != nil {
		return nil // user aborted before there's anything to lose
	}
	wizardLang = lang
	t := func(key string) string { return i18n.T(lang, key) }
	tf := func(key string, a ...any) string { return i18n.F(lang, key, a...) }
	cancelled := func() error {
		fmt.Println(styleWarn.Render(t("wizard.cancelled")))
		return nil
	}
	if firstRun {
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

	// ── 6. Existing config → action menu instead of a full re-walk ───────────
	// Re-running init against a config that already has profiles no longer
	// marches through every step again (which meant clicking "no" through
	// every other profile just to refresh one). It shows what's there and
	// drops into a menu of actions, repeated until the user saves or exits.
	if existing != nil && len(existing.VPNs) > 0 {
		return runEditMenu(lang, doc, existing, detected, outPath, home, tf, cancelled)
	}

	// ── 7-10. First-time setup: no existing profiles to choose among, so the
	// linear walk (profiles → policies → security → review) is the wizard. ──
	knownProfiles := make(map[string]bool)

	StepHeader(1, 4, t("section.profiles"))
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
		if err := addProfileFlow(lang, doc, detected, knownProfiles, &addedProfiles); err != nil {
			if errors.Is(err, errWizardCancelled) {
				return cancelled()
			}
			fmt.Println(tuiErr(err.Error()))
		}
	}

	addedPolicies, err := collectPoliciesHuh(lang, doc, existing, knownProfiles)
	if err != nil {
		if errors.Is(err, errWizardCancelled) {
			return cancelled()
		}
		return err
	}

	sec, err := collectSecurityHuh(lang, doc, home, securitySummary{
		killSwitch: true, dnsGuard: true, auditLog: true, monitor: true,
	})
	if err != nil {
		if errors.Is(err, errWizardCancelled) {
			return cancelled()
		}
		return err
	}

	if err := reviewAndWrite(lang, outPath, doc, addedProfiles, addedPolicies, sec); err != nil {
		if errors.Is(err, errWizardCancelled) {
			return cancelled()
		}
		return err
	}
	return nil
}

// selectWizardLanguage resolves the CLI's display language: from a saved
// preference, or by asking (and persisting) it on the very first run. Once a
// language is on record in preferences.json, every later `init` run (and
// every other command, via cliLang()) just uses it — changing it after that
// goes through `kongtrol config lang <es|en>`, not this wizard. Returns
// errWizardCancelled if the user aborts before anything is at risk.
func selectWizardLanguage() (i18n.Lang, bool, error) {
	prefs, prefsErr := loadPreferences()
	firstRun := prefsErr != nil || strings.TrimSpace(prefs.Language) == ""

	var langChoice string
	if firstRun {
		var choice string
		langForm := newForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Language / Idioma").
					Options(
						huh.NewOption("Español", "es"),
						huh.NewOption("English", "en"),
					).
					Value(&choice),
			),
		)
		if err := runForm(langForm); err != nil {
			return i18n.ES, firstRun, err
		}
		langChoice = choice
	} else {
		langChoice = strings.ToLower(strings.TrimSpace(prefs.Language))
	}

	lang := i18n.ES
	if langChoice == "en" {
		lang = i18n.EN
	}

	if firstRun {
		// Persist the choice so every other command (status, up, ...) uses it
		// too, and won't ask again.
		if p, err := loadPreferences(); err == nil {
			p.Language = langChoice
			_ = savePreferences(p)
		}
	}
	return lang, firstRun, nil
}

// printExistingProfiles lists every profile already in the loaded config,
// sorted by name, ahead of dropping into the edit menu.
func printExistingProfiles(tf func(string, ...any) string, outPath string, existing *config.Config) {
	names := make([]string, 0, len(existing.VPNs))
	for name := range existing.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)

	SectionHeader(tf("existing.header", outPath, len(existing.VPNs)))
	for _, name := range names {
		v := existing.VPNs[name]
		fmt.Printf("    %s  %-16s  type=%s  host=%s\n",
			styleInfo.Render("·"),
			styleBright.Render(name),
			styleWarn.Render(v.Type),
			styleDim.Render(v.Host))
	}
	fmt.Println()
}

// addProfileFlow drives a single "add a VPN profile" pass — collecting the
// profile via collectProfileHuh, handling the "profile already exists"
// replace confirmation, and writing the result into doc. Returns
// errWizardCancelled if the user aborted at any point.
func addProfileFlow(lang i18n.Lang, doc *yaml.Node, detected []detectedVPN, knownProfiles map[string]bool, addedProfiles *[]profileSummary) error {
	t := func(key string) string { return i18n.T(lang, key) }
	tf := func(key string, a ...any) string { return i18n.F(lang, key, a...) }

	profile, vpnNode, err := collectProfileHuh(lang, detected, knownProfiles)
	if err != nil {
		return err
	}
	if profile == "" {
		return nil
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
			return err
		}
		if !replace {
			fmt.Println(styleDim.Render("  " + t("profile.replace_skipped")))
			return nil
		}
		removeMapping(vpnsNode, profile)
	}

	vpnsNode.Content = append(vpnsNode.Content, scalarNode(profile), vpnNode)
	knownProfiles[profile] = true
	*addedProfiles = append(*addedProfiles, profileSummary{
		name: profile,
		typ:  valueOf(vpnNode, "type"),
		host: valueOf(vpnNode, "host"),
	})
	return nil
}

// reviewAndWrite renders the pre-write summary, confirms, and persists doc to
// outPath. Returns errWizardCancelled if the user aborts the final confirm.
func reviewAndWrite(lang i18n.Lang, outPath string, doc *yaml.Node, profiles []profileSummary, policies []string, sec securitySummary) error {
	t := func(key string) string { return i18n.T(lang, key) }
	tf := func(key string, a ...any) string { return i18n.F(lang, key, a...) }

	StepHeader(4, 4, t("section.write"))
	fmt.Println(renderReviewPanel(lang, outPath, profiles, policies, sec))
	fmt.Println()

	var doWrite bool
	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(tf("write.confirm", styleWarn.Render(outPath))).
				Value(&doWrite),
		),
	)); err != nil {
		return err
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
