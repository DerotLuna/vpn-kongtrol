package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ── ANSI palette ──────────────────────────────────────────────────────────────

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"

	// Kong amber/gold palette
	cKong    = "\033[38;5;136m" // dark amber  — gorilla body
	cKongHi  = "\033[38;5;178m" // light gold  — highlights
	cKongEye = "\033[38;5;51m"  // bright cyan — glowing eyes

	// UI semantic colors
	cSuccess = "\033[38;5;82m"  // bright green
	cError   = "\033[38;5;196m" // bright red
	cWarn    = "\033[38;5;214m" // orange
	cInfo    = "\033[38;5;39m"  // sky blue
	cPrompt  = "\033[38;5;51m"  // cyan
	cDim     = "\033[38;5;245m" // gray
	cBright  = "\033[38;5;255m" // near-white
	cGold    = "\033[38;5;220m" // gold
)

// isTerminal is true when stdout is an interactive terminal.
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

// paint wraps s in an ANSI color code iff stdout is a terminal.
func paint(code, s string) string {
	if !isTerminal {
		return s
	}
	return code + s + ansiReset
}

func paintBold(code, s string) string { return paint(code+ansiBold, s) }

// ── Semantic helpers ──────────────────────────────────────────────────────────

func tuiOK(s string) string     { return paintBold(cSuccess, "✓") + "  " + s }
func tuiErr(s string) string    { return paintBold(cError, "✗") + "  " + s }
func tuiWarn(s string) string   { return paintBold(cWarn, "!") + "  " + s }
func tuiInfo(s string) string   { return paint(cInfo, "▸") + "  " + s }
func tuiLabel(s string) string  { return paintBold(cPrompt, s) }
func tuiDim(s string) string    { return paint(cDim, s) }
func tuiBright(s string) string { return paintBold(cBright, s) }

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
					paintBold(cInfo, spinFrames[i%len(spinFrames)]),
					paint(cDim, s.label))
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
	time.Sleep(90 * time.Millisecond) // let goroutine exit
	fmt.Printf("\r\033[2K")           // clear line
}

// ── Kong gorilla logo ─────────────────────────────────────────────────────────

// kongTitle returns "K O N G T R O L" with an amber→gold gradient per letter.
func kongTitle() string {
	if !isTerminal {
		return "       K O N G T R O L"
	}
	gradient := []string{
		"\033[38;5;208m", // K deep orange
		"\033[38;5;214m", // O orange
		"\033[38;5;220m", // N gold
		"\033[38;5;226m", // G bright yellow
		"\033[38;5;220m", // T gold
		"\033[38;5;214m", // R orange
		"\033[38;5;208m", // O deep orange
		"\033[38;5;202m", // L red-orange
	}
	var sb strings.Builder
	sb.WriteString("      ")
	for i, ch := range "KONGTROL" {
		sb.WriteString(ansiBold)
		sb.WriteString(gradient[i])
		sb.WriteString(string(ch))
		sb.WriteString(ansiReset)
		sb.WriteString(" ")
	}
	return sb.String()
}

// kongLogoLines returns the gorilla face as a slice of printable strings.
// leftEye / rightEye are single display-width Unicode chars (e.g. ◉, •, ─).
// eyeColor is the ANSI color applied to those characters.
func kongLogoLines(leftEye, rightEye, eyeColor string) []string {
	b := func(s string) string { return paintBold(cKong, s) }    // body
	h := func(s string) string { return paintBold(cKongHi, s) }  // socket highlight
	e := func(s string) string { return paintBold(eyeColor, s) } // eye chars

	return []string{
		b("   ██████████████████████████"),
		b("  ██▀▀                    ▀▀██"),
		b("  ██  ") + h("▄▄███▄▄") + b("       ") + h("▄▄███▄▄") + b("  ██"),
		b("  ██  ") + h("██") + e(" "+leftEye+" ") + h("██") + b("       ") + h("██") + e(" "+rightEye+" ") + h("██") + b("  ██"),
		b("  ██  ") + h("▀▀███▀▀") + b("       ") + h("▀▀███▀▀") + b("  ██"),
		b("  ██      ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄     ██"),
		b("  ██     ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀    ██"),
		b("  ██▄▄                        ▄▄██"),
		b("   ██████████████████████████"),
		"",
		kongTitle(),
		paint(cDim, "      Multi-VPN Orchestration"),
	}
}

func printLogoLines(lines []string) {
	for _, l := range lines {
		fmt.Println(l)
	}
}

func redrawLogoLines(n int, lines []string) {
	fmt.Printf("\033[%dA", n) // cursor up n lines
	for _, l := range lines {
		fmt.Printf("\r\033[2K%s\n", l) // clear line + reprint
	}
}

// AnimateLogo prints the Kong gorilla logo with a two-blink eye animation.
func AnimateLogo() {
	const (
		eyeOpen   = "◉"
		eyeHalf   = "•"
		eyeClosed = "─"
	)

	type frame struct {
		l, r, eyeColor string
		hold           time.Duration
	}

	// Each frame shows for `hold` before advancing.
	anim := []frame{
		{eyeOpen, eyeOpen, cKongEye, 700 * time.Millisecond},
		{eyeHalf, eyeHalf, cKongEye, 60 * time.Millisecond},
		{eyeClosed, eyeClosed, cKong, 130 * time.Millisecond}, // closed: eye color = body → disappears
		{eyeHalf, eyeHalf, cKongEye, 60 * time.Millisecond},
		{eyeOpen, eyeOpen, cKongEye, 900 * time.Millisecond},
		{eyeHalf, eyeHalf, cKongEye, 60 * time.Millisecond},
		{eyeClosed, eyeClosed, cKong, 130 * time.Millisecond},
		{eyeHalf, eyeHalf, cKongEye, 60 * time.Millisecond},
		{eyeOpen, eyeOpen, cKongEye, 0}, // final state
	}

	first := kongLogoLines(anim[0].l, anim[0].r, anim[0].eyeColor)
	printLogoLines(first)
	n := len(first)

	if !isTerminal {
		return
	}

	for i := 0; i < len(anim)-1; i++ {
		time.Sleep(anim[i].hold)
		next := kongLogoLines(anim[i+1].l, anim[i+1].r, anim[i+1].eyeColor)
		redrawLogoLines(n, next)
	}
}

// SectionHeader prints a colored section divider line.
func SectionHeader(s string) {
	fmt.Printf("\n%s\n", tuiInfo(paintBold(cBright, s)))
}
