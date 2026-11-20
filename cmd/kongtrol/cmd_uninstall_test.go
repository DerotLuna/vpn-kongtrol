package main

import "testing"

// The Run value "Kongtrol" is removed only while it starts the tray the
// script installed: the Windows installer's own value (another folder) and
// a value that only mentions the path stay (ADR-0075).
func TestRunValueIsFor(t *testing.T) {
	tray := `C:\Users\u\AppData\Local\kongtrol\bin\kongtrol-tray.exe`
	cases := []struct {
		data string
		want bool
	}{
		{`"` + tray + `"`, true},
		{tray, true},
		{`"C:\USERS\U\APPDATA\LOCAL\KONGTROL\BIN\KONGTROL-TRAY.EXE"`, true},
		{`"` + tray + `" --open /`, true},
		{`"C:\Users\u\AppData\Local\Programs\Kongtrol\kongtrol-tray.exe"`, false},
		{`cmd /c "` + tray + `"`, false},
		{``, false},
		{`""`, false},
	}
	for _, c := range cases {
		if got := runValueIsFor(c.data, tray); got != c.want {
			t.Errorf("runValueIsFor(%q)=%v, want %v", c.data, got, c.want)
		}
	}
}
