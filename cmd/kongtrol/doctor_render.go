package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func (d *doctor) section(name string) {
	d.sectionN = name
	if !outputJSON {
		fmt.Printf("\n  %s\n", styleBright.Render(name))
		fmt.Printf("  %s\n", styleDim.Render(doctorRule()))
	}
}

func (d *doctor) record(status, label, detail string) {
	d.checks = append(d.checks, doctorCheck{
		Section: d.sectionN,
		Label:   label,
		Status:  status,
		Detail:  detail,
	})
}

func (d *doctor) ok(label, detail string) {
	d.record("ok", label, detail)
	if !outputJSON {
		d.printCheckLine("OK", label, detail)
	}
}

func (d *doctor) warn(label, detail string) {
	d.warnings++
	d.record("warn", label, detail)
	if !outputJSON {
		d.printCheckLine("WARN", label, detail)
	}
}

func (d *doctor) fail(label, detail string) {
	d.failures++
	d.record("fail", label, detail)
	if !outputJSON {
		d.printCheckLine("ERROR", label, detail)
	}
}

func (d *doctor) printCheckLine(level, label, detail string) {
	lines := wrapText(detail, doctorDetailWidth())
	if len(lines) == 0 {
		lines = []string{""}
	}
	badge := renderLevelBadge(level)
	fmt.Printf("  %s  %-*s  %s\n", badge, checkWidth, label, lines[0])
	for i := 1; i < len(lines); i++ {
		fmt.Printf("  %s  %-*s  %s\n", styleDim.Render(sym("│", "|")), checkWidth, "", styleDim.Render(lines[i]))
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
