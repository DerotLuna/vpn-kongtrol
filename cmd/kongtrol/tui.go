package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// isTerminal reports whether stdout is interactive and suitable for animation.
func isTerminal() bool {
	if outputQuiet {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Styles and palette live in theme.go ("Signal Contour").

// ── Semantic helpers ──────────────────────────────────────────────────────────

func sym(pretty, plain string) string {
	if outputPlain {
		return plain
	}
	return pretty
}

func tuiOK(s string) string     { return styleOK.Render(sym("✓", "[OK]")) + "  " + s }
func tuiErr(s string) string    { return styleErr.Render(sym("✗", "[ERR]")) + "  " + s }
func tuiWarn(s string) string   { return styleWarn.Render(sym("!", "[WARN]")) + "  " + s }
func tuiInfo(s string) string   { return styleInfo.Render(">") + "  " + s }
func tuiLabel(s string) string  { return stylePrompt.Render(s) }
func tuiDim(s string) string    { return styleDim.Render(s) }
func tuiBright(s string) string { return styleBright.Render(s) }

// ── Spinner ───────────────────────────────────────────────────────────────────

// Braille-dot spinner — dots orbit inside the cell. (Arc glyphs ◜◠◝ looked
// nicer on paper but are missing from common terminal fonts.)
var (
	spinFramesFancy = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinFramesPlain = []string{"-", "\\", "|", "/"}
)

type spinner struct {
	label     string
	stop      chan struct{}
	startedAt time.Time
}

func newSpinner(label string) *spinner {
	return &spinner{label: label, stop: make(chan struct{})}
}

func (s *spinner) Start() {
	s.startedAt = time.Now()
	if outputQuiet {
		return
	}
	if !isTerminal() {
		fmt.Println(s.label + "...")
		return
	}
	frames := spinFramesFancy
	if outputPlain {
		frames = spinFramesPlain
	}
	go func() {
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				return
			default:
				elapsed := time.Since(s.startedAt).Round(time.Second)
				fmt.Printf("\r\033[2K%s %s",
					stylePrompt.Render(frames[i%len(frames)]),
					styleDim.Render(fmt.Sprintf("%s (%s)", s.label, elapsed)))
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *spinner) Stop() {
	if outputQuiet || !isTerminal() {
		return
	}
	close(s.stop)
	time.Sleep(90 * time.Millisecond)
	fmt.Printf("\r\033[2K")
}

// ── Kong wordmark + banner ────────────────────────────────────────────────────

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

// kongBannerLines is the site hero's ANSI-Shadow KONGTROL wordmark — the same
// motd the landing page shows on `ssh guest@kongtrol.tower`.
var kongBannerLines = []string{
	`██╗  ██╗ ██████╗ ███╗   ██╗ ██████╗ ████████╗██████╗  ██████╗ ██╗`,
	`██║ ██╔╝██╔═══██╗████╗  ██║██╔════╝ ╚══██╔══╝██╔══██╗██╔═══██╗██║`,
	`█████╔╝ ██║   ██║██╔██╗ ██║██║  ███╗   ██║   ██████╔╝██║   ██║██║`,
	`██╔═██╗ ██║   ██║██║╚██╗██║██║   ██║   ██║   ██╔══██╗██║   ██║██║`,
	`██║  ██╗╚██████╔╝██║ ╚████║╚██████╔╝   ██║   ██║  ██║╚██████╔╝███████╗`,
	`╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝`,
}

// bannerRamp is the vertical amber gradient — brightest at the base, like the
// site's phosphor glow.
var bannerRamp = []lipgloss.Color{
	colEmber2, colEmber3, colSignal, colSignal, colSignalHi, colEmber3,
}

func renderBannerLines(ramp func(i int) lipgloss.Color) []string {
	out := make([]string, len(kongBannerLines))
	for i, l := range kongBannerLines {
		out[i] = "  " + lipgloss.NewStyle().Foreground(ramp(i)).Bold(true).Render(l)
	}
	return out
}

func redrawLines(n int, lines []string) {
	fmt.Printf("\033[%dA", n)
	for _, l := range lines {
		fmt.Printf("\r\033[2K%s\n", l)
	}
}

// AnimateBanner prints the block wordmark with a CRT power-on: lines reveal
// top to bottom, flash bright once, then settle on the amber ramp. Falls back
// to the compact wordmark on narrow terminals and in --plain mode.
func AnimateBanner(subtitle, ver string) {
	if outputPlain || lipgloss.Width(kongBannerLines[0])+2 > terminalWidth() {
		line := kongTitle()
		if ver != "" {
			line += "  " + styleDim.Render(ver)
		}
		line += "  " + styleDim.Render(sym("·", "-")+"  "+subtitle)
		fmt.Println("  " + line)
		return
	}

	settled := renderBannerLines(func(i int) lipgloss.Color { return bannerRamp[i%len(bannerRamp)] })
	if !isTerminal() {
		for _, l := range settled {
			fmt.Println(l)
		}
	} else {
		for _, l := range settled {
			fmt.Println(l)
			time.Sleep(28 * time.Millisecond)
		}
		time.Sleep(90 * time.Millisecond)
		redrawLines(len(settled), renderBannerLines(func(int) lipgloss.Color { return colSignalHi }))
		time.Sleep(120 * time.Millisecond)
		redrawLines(len(settled), settled)
	}

	meta := styleDim.Render(sym("#", "#") + " " + subtitle)
	if ver != "" {
		meta += styleDim.Render(" · " + ver)
	}
	meta += " " + styleInfo.Render("[beta]")
	fmt.Println()
	fmt.Println("  " + meta)
}

// PrintHeader prints a compact single-line banner for non-wizard commands.
func PrintHeader(ver string) {
	if outputQuiet {
		return
	}
	fmt.Println()
	line := kongTitle()
	if ver != "" {
		line += "  " + styleDim.Render(ver)
	}
	line += " " + styleInfo.Render("[beta]")
	line += "  " + styleDim.Render(sym("·", "-")+"  "+ct("cli.header.subtitle"))
	fmt.Println("  " + line)
	// Thin separator rule under the header
	fmt.Println("  " + lipgloss.NewStyle().Foreground(colRule).Render(strings.Repeat("─", 62)))
	fmt.Println()
}

// SectionHeader prints a section divider marked with the brand contour ring.
func SectionHeader(s string) {
	fmt.Printf("\n  %s  %s\n", stylePrompt.Render(sym("◎", "#")), styleBright.Render(s))
}

// StepHeader prints a numbered wizard phase header: ◎ 2/4 · Seguridad
func StepHeader(step, total int, s string) {
	fmt.Printf("\n  %s %s %s\n",
		stylePrompt.Render(sym("◎", "#")),
		styleDim.Render(fmt.Sprintf("%d/%d ·", step, total)),
		styleBright.Render(s))
}

// NextStepsPanel renders the wizard's closing card: a rounded amber box with
// each step as a shell command — the terminal twin of the site's quickstart
// card. Entries are i18n lines shaped "command — description".
func NextStepsPanel(title string, entries []string) string {
	type row struct{ cmd, desc string }
	rows := make([]row, 0, len(entries))
	maxW := 0
	for _, e := range entries {
		parts := strings.SplitN(strings.TrimSpace(e), "—", 2)
		r := row{cmd: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			r.desc = strings.TrimSpace(parts[1])
		}
		if w := lipgloss.Width(r.cmd); w > maxW {
			maxW = w
		}
		rows = append(rows, r)
	}

	var b strings.Builder
	b.WriteString(stylePrompt.Render(sym("◎", "#")) + "  " + styleGold.Render(strings.Trim(strings.TrimSpace(title), ":")))
	b.WriteString("\n")
	for _, r := range rows {
		b.WriteString("\n" + stylePrompt.Render("$ ") + styleBright.Render(r.cmd))
		if r.desc != "" {
			b.WriteString(strings.Repeat(" ", maxW-lipgloss.Width(r.cmd)+2))
			b.WriteString(styleDim.Render(sym("·", "-") + " " + r.desc))
		}
	}

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colEmber2).
		Padding(0, 2).
		Render(b.String())
	return "  " + strings.ReplaceAll(box, "\n", "\n  ")
}

// ── Legacy shims ──────────────────────────────────────────────────────────────
// Thin wrappers so wizard.go and main.go continue to compile until each file
// is rewritten with Lip Gloss / huh in its own task. Colors live in theme.go.

func paint(color lipgloss.Color, s string) string {
	return lipgloss.NewStyle().Foreground(color).Render(s)
}

func paintBold(color lipgloss.Color, s string) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(s)
}
