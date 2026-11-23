package autostart

import (
	"encoding/binary"
	"html"
	"strings"
	"time"
)

// The pure parts of each OS's format, kept free of build tags so every
// platform's tests cover them.

// ── Windows: Explorer\StartupApproved\Run ───────────────────────────────

// approvedEnabled reads a StartupApproved value: 12 bytes whose first byte
// is even when the entry may run (02) and odd when the user turned it off
// (03, followed by the FILETIME of that change). No value means enabled.
// Microsoft does not document this key; the layout is the one Task
// Manager writes (observed on Windows 11, research note of ADR-0098).
func approvedEnabled(data []byte) bool {
	return len(data) == 0 || data[0]&1 == 0
}

// approvalValue is the StartupApproved value for on/off at now.
func approvalValue(on bool, now time.Time) []byte {
	v := make([]byte, 12)
	if on {
		v[0] = 0x02
		return v
	}
	v[0] = 0x03
	binary.LittleEndian.PutUint64(v[4:], fileTime(now))
	return v
}

// fileTime converts t to a Windows FILETIME (100 ns since 1601-01-01 UTC).
func fileTime(t time.Time) uint64 {
	const epochDiff = 116444736000000000 // 1601→1970 in 100 ns units
	return uint64(t.UnixNano()/100) + epochDiff
}

// runValueFor is the Run value data for tray, quoted like the installer's.
func runValueFor(tray string) string { return `"` + tray + `"` }

// runValuePoints reports whether a Run value starts tray (quoted or not,
// any case: Windows paths are case-insensitive).
func runValuePoints(data, tray string) bool {
	d := strings.TrimSpace(data)
	if strings.HasPrefix(d, `"`) {
		if end := strings.Index(d[1:], `"`); end >= 0 {
			d = d[1 : end+1]
		}
	}
	return strings.EqualFold(d, tray)
}

// ── Linux: XDG Autostart desktop entry ─────────────────────────────────

// entryEnabled reports whether an autostart desktop entry starts anything:
// Hidden=true (XDG Autostart spec §2.2) and GNOME's
// X-GNOME-Autostart-enabled=false both turn it off.
func entryEnabled(content string) bool {
	hidden, gnomeOff := false, false
	forEachKey(content, func(key, value string) {
		switch key {
		case "Hidden":
			hidden = strings.EqualFold(value, "true")
		case "X-GNOME-Autostart-enabled":
			gnomeOff = strings.EqualFold(value, "false")
		}
	})
	return !hidden && !gnomeOff
}

// entryHasExec reports whether the entry can start a program on its own
// (an override written only to hide the system entry has no Exec).
func entryHasExec(content string) bool {
	has := false
	forEachKey(content, func(key, value string) {
		if key == "Exec" && strings.TrimSpace(value) != "" {
			has = true
		}
	})
	return has
}

// entryExec returns the entry's Exec value ("" when none).
func entryExec(content string) string {
	exec := ""
	forEachKey(content, func(key, value string) {
		if key == "Exec" {
			exec = strings.TrimSpace(value)
		}
	})
	return exec
}

// withHidden returns content with Hidden set as asked in its
// [Desktop Entry] group (and GNOME's switch removed, so one key decides).
func withHidden(content string, hidden bool) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+1)
	inMain := false
	insertAt := -1
	// endOfGroup is where the main group's last key line ends (before
	// trailing blank lines).
	endOfGroup := func() int {
		at := len(out)
		for at > 0 && strings.TrimSpace(out[at-1]) == "" {
			at--
		}
		return at
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			if inMain && insertAt < 0 {
				insertAt = endOfGroup()
			}
			inMain = trimmed == "[Desktop Entry]"
			out = append(out, line)
			continue
		}
		if inMain {
			key, _, ok := strings.Cut(trimmed, "=")
			key = strings.TrimSpace(key)
			if ok && (key == "Hidden" || key == "X-GNOME-Autostart-enabled") {
				continue
			}
		}
		out = append(out, line)
	}
	if inMain && insertAt < 0 {
		insertAt = endOfGroup() // the main group runs to the end
	}
	if hidden {
		if insertAt < 0 {
			out = append([]string{"[Desktop Entry]", "Hidden=true"}, out...)
		} else {
			out = append(out[:insertAt], append([]string{"Hidden=true"}, out[insertAt:]...)...)
		}
	}
	return strings.Join(out, "\n")
}

// forEachKey calls fn for every key=value of the [Desktop Entry] group.
func forEachKey(content string, fn func(key, value string)) {
	inMain := false
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inMain = trimmed == "[Desktop Entry]"
			continue
		}
		if !inMain {
			continue
		}
		if key, value, ok := strings.Cut(trimmed, "="); ok {
			fn(strings.TrimSpace(key), strings.TrimSpace(value))
		}
	}
}

// hiddenOverride hides the system entry for this user only (spec §2.3).
const hiddenOverride = "[Desktop Entry]\nType=Application\nName=Kongtrol\nHidden=true\n"

// desktopEntryFor is a full autostart entry that starts tray. Exec quotes
// the path when needed (Desktop Entry spec, "The Exec key").
func desktopEntryFor(tray string) string {
	return "[Desktop Entry]\nType=Application\nName=Kongtrol\n" +
		"Comment=Connect and control your VPNs from the system tray\n" +
		"Comment[es]=Conecta y controla tus VPN desde la bandeja del sistema\n" +
		"Exec=" + desktopExecQuote(tray) + "\nIcon=kongtrol\nTerminal=false\n"
}

func desktopExecQuote(path string) string {
	if !strings.ContainsAny(path, " \t\"'\\$`") {
		return path
	}
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`)
	return `"` + r.Replace(path) + `"`
}

// execStarts reports whether an Exec value starts tray (first argument,
// quoted or not).
func execStarts(exec, tray string) bool {
	e := strings.TrimSpace(exec)
	if strings.HasPrefix(e, `"`) {
		if end := strings.Index(e[1:], `"`); end >= 0 {
			e = e[1 : end+1]
		}
	} else if i := strings.IndexAny(e, " \t"); i >= 0 {
		e = e[:i]
	}
	return e == tray || e == desktopExecQuote(tray)
}

// ── macOS: LaunchAgent ──────────────────────────────────────────────────

// launchAgentLabel names Kongtrol's per-user LaunchAgent.
const launchAgentLabel = "app.kongtrol.tray"

// launchAgentFor is the LaunchAgent plist that opens Kongtrol at sign-in:
// the app bundle through open(1) when the tray lives inside one, else the
// tray binary itself. RunAtLoad starts it once when the agent loads at
// login (launchd.plist(5)).
func launchAgentFor(tray string) string {
	args := []string{tray}
	if app := appBundleOf(tray); app != "" {
		args = []string{"/usr/bin/open", "-a", app}
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>" + launchAgentLabel + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, a := range args {
		b.WriteString("\t\t<string>" + html.EscapeString(a) + "</string>\n")
	}
	b.WriteString("\t</array>\n\t<key>RunAtLoad</key>\n\t<true/>\n</dict>\n</plist>\n")
	return b.String()
}

// appBundleOf returns the .app folder holding tray (…/Kongtrol.app/
// Contents/MacOS/kongtrol-tray), or "".
func appBundleOf(tray string) string {
	const marker = ".app/Contents/MacOS/"
	if i := strings.Index(tray, marker); i >= 0 {
		return tray[:i+len(".app")]
	}
	return ""
}
