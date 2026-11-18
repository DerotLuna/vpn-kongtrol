package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/vpn-kongtrol/kongtrol/internal/doctor"
)

const checkWidth = 36

func printDoctorSection(name string) {
	fmt.Printf("\n  %s\n", styleBright.Render(name))
	fmt.Printf("  %s\n", styleDim.Render(doctorRule()))
}

// printDoctorCheck prints one check; warnings and errors add their
// "what to do" line underneath.
func printDoctorCheck(c doctor.Check) {
	level := "OK"
	switch c.Status {
	case doctor.StatusWarn:
		level = "WARN"
	case doctor.StatusFail:
		level = "ERROR"
	}
	lines := wrapText(c.Detail, doctorDetailWidth())
	if len(lines) == 0 {
		lines = []string{""}
	}
	fmt.Printf("  %s  %-*s  %s\n", renderLevelBadge(level), checkWidth, c.Label, lines[0])
	cont := styleDim.Render(sym("│", "|"))
	for _, l := range lines[1:] {
		fmt.Printf("  %s  %-*s  %s\n", cont, checkWidth, "", styleDim.Render(l))
	}
	if c.Hint == "" || c.Status == doctor.StatusOK {
		return
	}
	for i, l := range wrapText(c.Hint, doctorDetailWidth()-2) {
		mark := "  "
		if i == 0 {
			mark = styleInfo.Render(sym("→", ">")) + " "
		}
		fmt.Printf("  %s  %-*s  %s%s\n", cont, checkWidth, "", mark, l)
	}
}

func doctorRule() string {
	w := terminalWidth() - 4
	if w < 32 {
		w = 32
	}
	return strings.Repeat("─", w)
}

func doctorDetailWidth() int {
	// "  <mark>  <label(36)>  <detail>"
	w := terminalWidth() - (2 + 3 + 2 + checkWidth + 2)
	if w < 24 {
		w = 24
	}
	return w
}

func wrapText(s string, width int) []string {
	if width <= 0 || s == "" {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, 4)
	current := words[0]
	for _, w := range words[1:] {
		candidate := current + " " + w
		if utf8.RuneCountInString(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		if utf8.RuneCountInString(w) <= width {
			current = w
			continue
		}
		// hard-wrap long tokens (paths/URLs)
		for utf8.RuneCountInString(w) > width {
			lines = append(lines, truncateRunes(w, width))
			w = trimRunesPrefix(w, width)
		}
		current = w
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func trimRunesPrefix(s string, n int) string {
	if n <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return ""
	}
	return string(r[n:])
}
