package autostart

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func TestApprovedEnabled(t *testing.T) {
	// 03 + FILETIME is what Task Manager writes for a disabled entry
	// (seen on Windows 11); 02 + zeros for an enabled one.
	for _, c := range []struct {
		data []byte
		want bool
	}{
		{nil, true},
		{[]byte{0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, true},
		{[]byte{0x06, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, true},
		{[]byte{0x03, 0, 0, 0, 0x7A, 0x34, 0x2B, 0x48, 0xA9, 0x0E, 0xDB, 0x01}, false},
	} {
		if got := approvedEnabled(c.data); got != c.want {
			t.Errorf("approvedEnabled(% x) = %v, want %v", c.data, got, c.want)
		}
	}
}

func TestApprovalValue(t *testing.T) {
	on := approvalValue(true, time.Now())
	if len(on) != 12 || on[0] != 0x02 || !approvedEnabled(on) {
		t.Errorf("on = % x", on)
	}
	when := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	off := approvalValue(false, when)
	if len(off) != 12 || off[0] != 0x03 || approvedEnabled(off) {
		t.Errorf("off = % x", off)
	}
	ft := binary.LittleEndian.Uint64(off[4:])
	back := time.Unix(0, int64(ft-116444736000000000)*100).UTC()
	if !back.Equal(when) {
		t.Errorf("FILETIME round trip = %v, want %v", back, when)
	}
}

func TestRunValuePoints(t *testing.T) {
	tray := `C:\Users\ana\AppData\Local\Programs\Kongtrol\kongtrol-tray.exe`
	if !runValuePoints(runValueFor(tray), tray) || !runValuePoints(strings.ToUpper(tray), tray) {
		t.Error("the installer's value was not recognised")
	}
	if runValuePoints(`"C:\Other\kongtrol-tray.exe"`, tray) {
		t.Error("another copy's value was taken as ours")
	}
}

const packaged = `[Desktop Entry]
# Kongtrol menu entry
Type=Application
Name=Kongtrol
Exec=/home/ana/.local/bin/kongtrol-tray
Icon=kongtrol
Terminal=false

[Desktop Action quit]
Name=Quit
Exec=kongtrol-tray --quit
`

func TestWithHidden(t *testing.T) {
	off := withHidden(packaged, true)
	if entryEnabled(off) {
		t.Fatalf("Hidden=true not applied:\n%s", off)
	}
	if !strings.Contains(off, "Terminal=false\nHidden=true\n\n[Desktop Action quit]") {
		t.Errorf("Hidden must go in [Desktop Entry], before the next group:\n%s", off)
	}
	if got := withHidden(withHidden(off, true), true); strings.Count(got, "Hidden=") != 1 {
		t.Errorf("Hidden written twice:\n%s", got)
	}
	on := withHidden(off, false)
	if !entryEnabled(on) || on != packaged {
		t.Errorf("turning it back on did not restore the file:\n%s", on)
	}
	gnome := withHidden("[Desktop Entry]\nExec=x\nX-GNOME-Autostart-enabled=false\n", false)
	if !entryEnabled(gnome) {
		t.Errorf("GNOME's switch left the entry off:\n%s", gnome)
	}
	if !entryEnabled(withHidden("[Desktop Entry]\nExec=x", false)) || entryEnabled(withHidden("[Desktop Entry]\nExec=x", true)) {
		t.Error("an entry without a trailing newline")
	}
}

func TestEntryExecAndOverride(t *testing.T) {
	if entryHasExec(hiddenOverride) || entryEnabled(hiddenOverride) {
		t.Error("the override must only hide the system entry")
	}
	tray := "/home/ana/.local/bin/kongtrol-tray"
	if !execStarts(entryExec(packaged), tray) || execStarts(entryExec(packaged), "/usr/bin/kongtrol-tray") {
		t.Error("execStarts")
	}
	spaced := "/home/ana/My Apps/kongtrol-tray"
	entry := desktopEntryFor(spaced)
	if !strings.Contains(entry, `Exec="/home/ana/My Apps/kongtrol-tray"`) || !execStarts(entryExec(entry), spaced) {
		t.Errorf("a path with a space must be quoted:\n%s", entry)
	}
}

func TestLaunchAgentFor(t *testing.T) {
	app := launchAgentFor("/Applications/Kongtrol.app/Contents/MacOS/kongtrol-tray")
	for _, want := range []string{"<string>app.kongtrol.tray</string>", "<string>/usr/bin/open</string>",
		"<string>/Applications/Kongtrol.app</string>", "<key>RunAtLoad</key>\n\t<true/>"} {
		if !strings.Contains(app, want) {
			t.Errorf("missing %q:\n%s", want, app)
		}
	}
	bare := launchAgentFor("/opt/k&k/kongtrol-tray")
	if !strings.Contains(bare, "<string>/opt/k&amp;k/kongtrol-tray</string>") || strings.Contains(bare, "/usr/bin/open") {
		t.Errorf("bare binary:\n%s", bare)
	}
}
