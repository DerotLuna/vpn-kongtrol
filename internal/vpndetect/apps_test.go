package vpndetect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestVersionDirsNewestFirst(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"numeric not lexical", []string{"v4.9.0", "v4.10.1", "v4.1.10"}, []string{"v4.10.1", "v4.9.0", "v4.1.10"}},
		{"non-versions dropped", []string{"Resources", "v4.0.0", "en-us", "v", "vx.1"}, []string{"v4.0.0"}},
		{"no v prefix (NordVPN)", []string{"7.24.5.0", "7.3.0.0", "Drivers"}, []string{"7.24.5.0", "7.3.0.0"}},
		{"missing parts are zero", []string{"v4.1", "v4.1.0.1"}, []string{"v4.1.0.1", "v4.1"}},
		{"too many parts", []string{"1.2.3.4.5"}, []string{}},
		{"empty", nil, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionDirsNewestFirst(tt.in); !slices.Equal(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComparePaths(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{`C:\P\Proton\VPN\v4.10.0\ProtonVPN.Client.exe`, `C:\P\Proton\VPN\v4.9.0\ProtonVPN.Client.exe`, 1},
		{`C:\P\vendor\App.exe`, `c:\p\VENDOR\app.exe`, 0},
		{`/opt/a/b`, `/opt/a`, 1},
		{`/opt/a`, `/opt/b`, -1},
	}
	for _, tt := range tests {
		if got := comparePaths(tt.a, tt.b); sign(got) != tt.want {
			t.Errorf("comparePaths(%q, %q) = %d, want sign %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func TestInstallDirFromEntry(t *testing.T) {
	tests := []struct {
		name, loc, icon, want string
	}{
		{"install location wins", `C:\Program Files\Proton\VPN\`, `C:\Other\x.exe`, `C:\Program Files\Proton\VPN`},
		{"quoted location", `"D:\Apps\NordVPN"`, "", `D:\Apps\NordVPN`},
		{"icon with index", "", `"C:\Program Files\Proton\VPN\ProtonVPN.Launcher.exe",0`, `C:\Program Files\Proton\VPN`},
		{"icon plain", "", `C:\Program Files (x86)\Surfshark\Surfshark.exe`, `C:\Program Files (x86)\Surfshark`},
		{"relative ignored", `Proton\VPN`, `ProtonVPN.exe`, ""},
		{"nothing", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InstallDirFromEntry(tt.loc, tt.icon); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAppPathsProtonLayout(t *testing.T) {
	root := t.TempDir()
	vpnDir := filepath.Join(root, "Proton", "VPN")
	launcher := filepath.Join(vpnDir, "ProtonVPN.Launcher.exe")
	newest := filepath.Join(vpnDir, "v4.10.2", "ProtonVPN.Client.exe")
	touch(t, launcher)
	touch(t, filepath.Join(vpnDir, "v4.9.0", "ProtonVPN.Client.exe"))
	touch(t, newest)
	touch(t, filepath.Join(vpnDir, "v4.10.2", "Resources", "openvpn.exe"))

	layout := AppLayout{Exes: []string{"ProtonVPN.Launcher.exe"}, VersionedExes: []string{"ProtonVPN.Client.exe"}}
	got := AppPaths([]string{"", filepath.Join(root, "missing"), vpnDir, vpnDir}, layout)
	want := []string{launcher, newest, filepath.Join(vpnDir, "v4.9.0", "ProtonVPN.Client.exe")}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}

	// Without the launcher the newest versioned client comes first.
	if err := os.Remove(launcher); err != nil {
		t.Fatal(err)
	}
	if got := AppPaths([]string{vpnDir}, layout); len(got) == 0 || got[0] != newest {
		t.Fatalf("got %q, want %q first", got, newest)
	}
}

func TestDetectCandidatesAndNewestVersion(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "App.exe")
	touch(t, exe)
	p := Probe{Label: "x", Candidates: func() []string { return []string{filepath.Join(root, "nope.exe"), "relative.exe", exe} }}
	if got, ok := Detect(p); !ok || got != exe {
		t.Fatalf("Detect = %q, %v; want %q", got, ok, exe)
	}

	// SearchDirs: v4.10 must beat v4.9 (a string sort picks v4.9).
	old := filepath.Join(root, "Vendor", "v4.9.0", "client.exe")
	newer := filepath.Join(root, "Vendor", "v4.10.0", "client.exe")
	touch(t, old)
	touch(t, newer)
	p = Probe{Label: "y", SearchDirs: []string{filepath.Join(root, "Vendor")}, SearchDepth: 1, ExeNames: []string{"client.exe"}}
	if got, ok := Detect(p); !ok || got != newer {
		t.Fatalf("Detect = %q, %v; want %q", got, ok, newer)
	}
}
