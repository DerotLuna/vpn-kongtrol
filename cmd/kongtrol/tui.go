package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// isTerminal is true when stdout is an interactive terminal.
// Used only for cursor/animation control — Lip Gloss handles color detection
// automatically via termenv.
var isTerminal = func() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}()

// ── Lip Gloss styles ──────────────────────────────────────────────────────────

var (
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	styleInfo   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	stylePrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleBright = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleGold   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	styleKong   = lipgloss.NewStyle().Foreground(lipgloss.Color("136")).Bold(true)
	styleEye    = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)

	// Block-char shading styles — amber gradient: ░ → ▒ → ▓ → █
	shadeLight = lipgloss.NewStyle().Foreground(lipgloss.Color("94"))  // dark olive (outermost)
	shadeMed   = lipgloss.NewStyle().Foreground(lipgloss.Color("130")) // brown
	shadeHeavy = lipgloss.NewStyle().Foreground(lipgloss.Color("136")) // amber
	shadeSolid = lipgloss.NewStyle().Foreground(lipgloss.Color("172")) // bright amber (core)
)

// ── Title gradient ─────────────────────────────────────────────────────────────

// titleGradient holds per-letter ANSI 256 colors: K O N G T R O L
var titleGradient = []lipgloss.Color{"208", "214", "220", "226", "220", "214", "208", "202"}

// kongShade colors each ░▒▓█ rune with its corresponding amber shade.
// All other characters are passed through unstyled.
func kongShade(s string) string {
	var sb strings.Builder
	for _, ch := range s {
		switch ch {
		case '░':
			sb.WriteString(shadeLight.Render(string(ch)))
		case '▒':
			sb.WriteString(shadeMed.Render(string(ch)))
		case '▓':
			sb.WriteString(shadeHeavy.Render(string(ch)))
		case '█':
			sb.WriteString(shadeSolid.Render(string(ch)))
		default:
			sb.WriteString(string(ch))
		}
	}
	return sb.String()
}

// ── Semantic helpers ──────────────────────────────────────────────────────────

func tuiOK(s string) string     { return styleOK.Render("✓") + "  " + s }
func tuiErr(s string) string    { return styleErr.Render("✗") + "  " + s }
func tuiWarn(s string) string   { return styleWarn.Render("!") + "  " + s }
func tuiInfo(s string) string   { return styleInfo.Render("▸") + "  " + s }
func tuiLabel(s string) string  { return stylePrompt.Render(s) }
func tuiDim(s string) string    { return styleDim.Render(s) }
func tuiBright(s string) string { return styleBright.Render(s) }

// ── Spinner ───────────────────────────────────────────────────────────────────

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinner struct {
	label string
	stop  chan struct{}
}

func newSpinner(label string) *spinner {
	return &spinner{label: label, stop: make(chan struct{})}
}

func (s *spinner) Start() {
	if !isTerminal {
		fmt.Println(s.label + "...")
		return
	}
	go func() {
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r\033[2K%s %s",
					styleInfo.Render(spinFrames[i%len(spinFrames)]),
					styleDim.Render(s.label))
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *spinner) Stop() {
	if !isTerminal {
		return
	}
	close(s.stop)
	time.Sleep(90 * time.Millisecond)
	fmt.Printf("\r\033[2K")
}

// ── Kong gorilla logo ─────────────────────────────────────────────────────────

// kongTitle returns "K O N G T R O L" with an amber→gold gradient per letter.
func kongTitle() string {
	bold := lipgloss.NewStyle().Bold(true)
	var sb strings.Builder
	for i, ch := range "KONGTROL" {
		sb.WriteString(bold.Foreground(titleGradient[i]).Render(string(ch)))
		if i < 7 {
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

// kongLogoLines returns a 9-line compact gorilla face using ░▒▓█ block chars
// with an amber gradient, animated eyes, and text to the right.
//
//	  ▲  ▲  ▲  ▲                             (crown — animated gold spikes)
//	  ░▒▓██████████▓▒░                        (top of head)
//	 ▒▓████▒░    ░▒████▓▒                     (temples)
//	▒▓████  ░▒▓░  ░▓▒░  ████▓▒               (brow ridge)
//	▓████  ░█◉█░  ░█◉█░  ████▓  K O N G T R O L
//	▒████   ░░      ░░   ████▒  Subtitle
//	░████  ░▓████████▓░  ████░  v1.2.3
//	 ▒▓████▒░      ░▒████▓▒                   (chin)
//	  ░▒▒▓████████████▓▒▒░                    (base)
//
// Crown spikes and eyes animate in sync.
func kongLogoLines(leftEye, rightEye string, eyeStyle lipgloss.Style, crownStyle lipgloss.Style, subtitle, ver string) []string {
	s := kongShade
	e := eyeStyle.Render
	k := crownStyle.Render

	titleText := "  " + kongTitle()
	subText := ""
	if subtitle != "" {
		subText = "  " + styleDim.Render(subtitle)
	}
	verText := ""
	if ver != "" {
		verText = "  " + styleDim.Render(ver)
	}

	return []string{
		"  " + k(" ▲  ▲  ▲  ▲ "),
		"  " + s("░▒▓██████████▓▒░"),
		" " + s("▒▓████▒░    ░▒████▓▒"),
		s("▒▓████  ░▒▓░  ░▓▒░  ████▓▒"),
		s("▓████  ░█") + e(leftEye) + s("█░  ░█") + e(rightEye) + s("█░  ████▓") + titleText,
		s("▒████   ░░      ░░   ████▒") + subText,
		s("░████  ░▓████████▓░  ████░") + verText,
		" " + s("▒▓████▒░      ░▒████▓▒"),
		"  " + s("░▒▒▓████████████▓▒▒░"),
	}
}

func printLogoLines(lines []string) {
	for _, l := range lines {
		fmt.Println(l)
	}
}

func redrawLogoLines(n int, lines []string) {
	fmt.Printf("\033[%dA", n)
	for _, l := range lines {
		fmt.Printf("\r\033[2K%s\n", l)
	}
}

// PrintHeader prints a compact single-line banner for non-wizard commands.
func PrintHeader(ver string) {
	fmt.Println()
	crown := styleGold.Render("▲ ▲ ▲ ▲")
	line := crown + "  " + kongTitle()
	if ver != "" {
		line += "  " + styleDim.Render(ver)
	}
	line += "  " + styleDim.Render("·  Multi-VPN Orchestration")
	fmt.Println("  " + line)
	// Thin separator rule under the header
	fmt.Println("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Render(strings.Repeat("─", 62)))
	fmt.Println()
}

// AnimateLogo prints the Kong logo with a two-blink eye animation.
// The crown spikes glow/dim/vanish in sync with the eye state.
func AnimateLogo(subtitle, ver string) {
	const (
		eyeOpen   = "◉"
		eyeHalf   = "•"
		eyeClosed = "─"
	)

	crownGlow  := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true) // bright gold
	crownDim   := lipgloss.NewStyle().Foreground(lipgloss.Color("136"))            // amber (dim)
	crownGone  := styleKong                                                        // same as body → invisible

	type frame struct {
		l, r        string
		eyeStyle    lipgloss.Style
		crownStyle  lipgloss.Style
		hold        time.Duration
	}

	anim := []frame{
		{eyeOpen,   eyeOpen,   styleEye,  crownGlow, 700 * time.Millisecond},
		{eyeHalf,   eyeHalf,   styleEye,  crownDim,  60 * time.Millisecond},
		{eyeClosed, eyeClosed, styleKong, crownGone, 130 * time.Millisecond},
		{eyeHalf,   eyeHalf,   styleEye,  crownDim,  60 * time.Millisecond},
		{eyeOpen,   eyeOpen,   styleEye,  crownGlow, 900 * time.Millisecond},
		{eyeHalf,   eyeHalf,   styleEye,  crownDim,  60 * time.Millisecond},
		{eyeClosed, eyeClosed, styleKong, crownGone, 130 * time.Millisecond},
		{eyeHalf,   eyeHalf,   styleEye,  crownDim,  60 * time.Millisecond},
		{eyeOpen,   eyeOpen,   styleEye,  crownGlow, 0}, // final state
	}

	first := kongLogoLines(anim[0].l, anim[0].r, anim[0].eyeStyle, anim[0].crownStyle, subtitle, ver)
	printLogoLines(first)
	n := len(first)

	if !isTerminal {
		return
	}

	for i := 0; i < len(anim)-1; i++ {
		time.Sleep(anim[i].hold)
		f := anim[i+1]
		next := kongLogoLines(f.l, f.r, f.eyeStyle, f.crownStyle, subtitle, ver)
		redrawLogoLines(n, next)
	}
}

// SectionHeader prints a colored section divider line.
func SectionHeader(s string) {
	fmt.Printf("\n%s\n", tuiInfo(styleBright.Render(s)))
}

// ── Legacy shims ──────────────────────────────────────────────────────────────
// Thin wrappers so wizard.go and main.go continue to compile until each file
// is rewritten with Lip Gloss / huh in its own task.

var (
	cSuccess = lipgloss.Color("82")
	cError   = lipgloss.Color("196")
	cWarn    = lipgloss.Color("214")
	cInfo    = lipgloss.Color("39")
	cPrompt  = lipgloss.Color("51")
	cDim     = lipgloss.Color("245")
	cBright  = lipgloss.Color("255")
	cGold    = lipgloss.Color("220")
)

func paint(color lipgloss.Color, s string) string {
	return lipgloss.NewStyle().Foreground(color).Render(s)
}

func paintBold(color lipgloss.Color, s string) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(s)
}
