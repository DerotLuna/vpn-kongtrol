package uninstall

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPurge_EnumeratesExactlyKongtrolCredentials(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "kongtrol.yaml"), "vpns:\n  work:\n    type: openvpn\n  home:\n    type: wireguard\n")
	write(t, filepath.Join(dir, "kongtrol.yaml.bak-2026-11-17"), "vpns:\n  old:\n    type: openvpn\n  work:\n    type: openvpn\n")
	write(t, filepath.Join(dir, "broken.yaml"), "vpns: [::")
	elsewhere := filepath.Join(t.TempDir(), "custom.yaml")
	write(t, elsewhere, "vpns:\n  lab:\n    type: openvpn\n")

	var deleted []string
	rep := Purge(PurgeOptions{
		DataDir:      dir,
		ExtraFiles: []string{elsewhere},
		Delete: func(c Credential) error {
			deleted = append(deleted, c.String())
			return nil
		},
	})
	sort.Strings(deleted)
	want := []string{
		"_system.audit_hmac_key",
		"home.password", "home.username",
		"lab.password", "lab.username",
		"old.password", "old.username",
		"work.password", "work.username",
	}
	if !reflect.DeepEqual(deleted, want) {
		t.Fatalf("deleted %v\nwant    %v", deleted, want)
	}
	if rep.Err() != nil {
		t.Fatal(rep.Err())
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("data dir must be removed")
	}
	if _, err := os.Stat(elsewhere); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("--config file outside the data dir must be removed")
	}
}

func TestPurge_DryRunTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "kongtrol.yaml"), "vpns:\n  work: {type: openvpn}\n")
	rep := Purge(PurgeOptions{DataDir: dir, DryRun: true, Delete: func(Credential) error {
		t.Fatal("dry run must not delete credentials")
		return nil
	}})
	if len(rep.Credentials) != 3 || len(rep.Paths) != 1 {
		t.Fatalf("unexpected dry-run report %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dir, "kongtrol.yaml")); err != nil {
		t.Fatal("dry run removed a file")
	}
}

func TestPurge_KeepRuntime(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"kongtrol.yaml", "preferences.json", "history.json", "api-token", "daemon.log", ManifestName, "run/kongtrol.pid", "wireguard/x.conf"} {
		write(t, filepath.Join(dir, f), "vpns: {}\n")
	}
	rep := Purge(PurgeOptions{DataDir: dir, KeepRuntime: true, Delete: func(Credential) error { return nil }})
	if rep.Err() != nil {
		t.Fatal(rep.Err())
	}
	for _, keep := range []string{"api-token", "daemon.log", ManifestName, "run"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("%s must be kept", keep)
		}
	}
	for _, gone := range []string{"kongtrol.yaml", "preferences.json", "history.json", "wireguard"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); err == nil {
			t.Errorf("%s must be deleted", gone)
		}
	}
}

func TestPurge_KeychainFailureReported(t *testing.T) {
	dir := t.TempDir()
	rep := Purge(PurgeOptions{DataDir: dir, Delete: func(Credential) error { return errors.New("locked") }})
	if rep.Err() == nil || len(rep.Credentials) != 0 {
		t.Fatalf("failure must be reported: %+v", rep)
	}
}

func TestDetect_PerChannel(t *testing.T) {
	data := t.TempDir()
	none := func(string) bool { return false }

	t.Run("script via manifest", func(t *testing.T) {
		d := t.TempDir()
		bin := filepath.Join(d, "bin", "kongtrol")
		write(t, filepath.Join(d, ManifestName), "# kongtrol install.sh\nfile="+bin+"\nfile="+filepath.Join(d, "bin", "kongtrol-tray")+"\nautostart="+filepath.Join(d, "kongtrol.desktop")+"\nfile=relative/ignored\n")
		p := Detect(Env{GOOS: "linux", Exe: bin, DataDir: d, Exists: none}, true, false)
		if p.Channel != ChannelScript || len(p.Files) != 2 || len(p.Autostart) != 1 || !p.Purge {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("script manifest of another copy is ignored", func(t *testing.T) {
		d := t.TempDir()
		write(t, filepath.Join(d, ManifestName), "file="+filepath.Join(d, "other", "kongtrol")+"\n")
		p := Detect(Env{GOOS: "linux", Exe: "/home/u/src/kongtrol", DataDir: d, Exists: none}, false, false)
		if p.Channel != ChannelManual || len(p.Files) != 0 {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("macOS app bundle via manifest", func(t *testing.T) {
		d := t.TempDir()
		app := filepath.Join(d, "Applications", "Kongtrol.app")
		write(t, filepath.Join(d, ManifestName), "file="+app+"\n")
		p := Detect(Env{GOOS: "darwin", Exe: filepath.Join(app, "Contents", "MacOS", "kongtrol"), DataDir: d, Exists: none}, false, false)
		if p.Channel != ChannelScript {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("windows installer", func(t *testing.T) {
		exe := `C:\Users\u\AppData\Local\Programs\Kongtrol\kongtrol.exe`
		p := Detect(Env{GOOS: "windows", Exe: exe, DataDir: data, Exists: func(s string) bool {
			return filepath.Base(s) == "unins000.exe"
		}}, false, false)
		if p.Channel != ChannelWindowsInstaller || p.Uninstaller == "" || len(p.Files) != 0 {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("legacy install.ps1 location", func(t *testing.T) {
		lad := filepath.Join(t.TempDir(), "Local")
		exe := filepath.Join(lad, "kongtrol", "bin", "kongtrol.exe")
		p := Detect(Env{GOOS: "windows", Exe: exe, DataDir: data, LocalAppData: lad, Exists: none}, false, false)
		if p.Channel != ChannelScript || len(p.Files) != 1 || len(p.PathEntries) != 1 {
			t.Fatalf("got %+v", p)
		}
	})
	for _, tc := range []struct{ goos, exe, mgr string }{
		{"darwin", "/opt/homebrew/Cellar/kongtrol/1.0.0/bin/kongtrol", "brew"},
		{"linux", "/home/linuxbrew/.linuxbrew/Cellar/kongtrol/1.0.0/bin/kongtrol", "brew"},
		{"linux", "/usr/lib/node_modules/@kongtrol/cli-linux-x64/bin/kongtrol", "npm"},
		{"linux", "/usr/bin/kongtrol", "system"},
		{"windows", `C:\Users\u\scoop\apps\kongtrol\current\kongtrol.exe`, "scoop"},
	} {
		t.Run("package manager "+tc.mgr, func(t *testing.T) {
			p := Detect(Env{GOOS: tc.goos, Exe: tc.exe, DataDir: data, Exists: none}, true, false)
			if p.Channel != ChannelPackageManager || p.Manager != tc.mgr || len(p.Files) != 0 {
				t.Fatalf("package-manager files must never be removed: %+v", p)
			}
		})
	}
	t.Run("manual", func(t *testing.T) {
		p := Detect(Env{GOOS: "linux", Exe: "/home/u/go/bin/kongtrol", DataDir: data, Exists: none}, false, false)
		if p.Channel != ChannelManual || len(p.Files) != 0 {
			t.Fatalf("got %+v", p)
		}
	})
	t.Run("data only implies purge", func(t *testing.T) {
		p := Detect(Env{GOOS: "linux", Exe: "/x/kongtrol", DataDir: data, Exists: none}, false, true)
		if !p.Purge || !p.DataOnly {
			t.Fatalf("got %+v", p)
		}
	})
}
