package uninstall

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Channel is how this copy of Kongtrol was installed. Only files Kongtrol's
// own install scripts put there are ever deleted by `kongtrol uninstall`.
type Channel string

const (
	// ChannelScript: install.sh / install.ps1 (listed in install.manifest).
	ChannelScript Channel = "script"
	// ChannelWindowsInstaller: the Inno Setup installer (unins000.exe next
	// to kongtrol.exe). Its own uninstaller removes the files.
	ChannelWindowsInstaller Channel = "windows-installer"
	// ChannelPackageManager: Homebrew, npm, Scoop, WinGet or the system
	// package manager own the files; only they may remove them.
	ChannelPackageManager Channel = "package-manager"
	// ChannelManual: unknown origin (copied by hand, built from source).
	ChannelManual Channel = "manual"
)

// Plan is what `kongtrol uninstall` will do. Build it with Detect; it is
// pure data so it can be printed (--dry-run) and tested per channel.
type Plan struct {
	Channel Channel `json:"channel"`
	// Manager names the package manager (brew, npm, scoop, winget, system).
	Manager string `json:"manager,omitempty"`
	Exe     string `json:"exe"`
	// Uninstaller is the Windows installer's unins000.exe.
	Uninstaller string `json:"uninstaller,omitempty"`
	// Files are removed only for ChannelScript (binaries, app bundle, menu
	// entry, icon).
	Files []string `json:"files,omitempty"`
	// Autostart entries Kongtrol's script created.
	Autostart []string `json:"autostart,omitempty"`
	// PathEntries are directories the script added to the user PATH
	// (Windows); removed from HKCU\Environment\Path.
	PathEntries []string `json:"path_entries,omitempty"`
	DataDir     string   `json:"data_dir"`
	Purge       bool     `json:"purge"`
	DataOnly    bool     `json:"data_only"`
}

// Manifest is the parsed install.manifest ("key=value" lines: file=,
// autostart=, path=).
type Manifest struct {
	Files       []string
	Autostart   []string
	PathEntries []string
}

// ReadManifest parses dataDir/install.manifest; ok is false when absent.
func ReadManifest(dataDir string) (Manifest, bool) {
	var m Manifest
	f, err := os.Open(filepath.Join(dataDir, ManifestName))
	if err != nil {
		return m, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), string(rune(0xFEFF))))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		v = strings.TrimSpace(v)
		if !ok || v == "" || !filepath.IsAbs(v) {
			continue // never act on a relative path
		}
		switch strings.TrimSpace(k) {
		case "file":
			m.Files = append(m.Files, filepath.Clean(v))
		case "autostart":
			m.Autostart = append(m.Autostart, filepath.Clean(v))
		case "path":
			m.PathEntries = append(m.PathEntries, filepath.Clean(v))
		}
	}
	return m, true
}

// Env is what Detect looks at; tests fill it by hand.
type Env struct {
	GOOS         string
	Exe          string // resolved path of the running kongtrol
	DataDir      string // ~/.kongtrol
	LocalAppData string // Windows %LOCALAPPDATA%
	// Exists reports whether a path exists (nil = os.Stat).
	Exists func(string) bool
}

func (e Env) exists(p string) bool {
	if e.Exists != nil {
		return e.Exists(p)
	}
	_, err := os.Stat(p)
	return err == nil
}

// Detect decides the channel and the exact files to remove.
func Detect(env Env, purge, dataOnly bool) Plan {
	p := Plan{Exe: env.Exe, DataDir: env.DataDir, Purge: purge || dataOnly, DataOnly: dataOnly}
	exe := filepath.Clean(env.Exe)
	norm := strings.ToLower(filepath.ToSlash(exe))
	dir := filepath.Dir(exe)

	if m, ok := ReadManifest(env.DataDir); ok && manifestOwns(m, exe) {
		p.Channel = ChannelScript
		p.Files, p.Autostart, p.PathEntries = m.Files, m.Autostart, m.PathEntries
		return p
	}
	if env.GOOS == "windows" {
		if un := filepath.Join(dir, "unins000.exe"); env.exists(un) {
			p.Channel, p.Uninstaller = ChannelWindowsInstaller, un
			return p
		}
	}
	if mgr := packageManager(env.GOOS, norm); mgr != "" {
		p.Channel, p.Manager = ChannelPackageManager, mgr
		return p
	}
	// install.ps1 before it wrote a manifest: its fixed default location.
	if env.GOOS == "windows" && env.LocalAppData != "" &&
		strings.EqualFold(dir, filepath.Join(env.LocalAppData, "kongtrol", "bin")) {
		p.Channel = ChannelScript
		p.Files = []string{exe}
		p.PathEntries = []string{dir}
		return p
	}
	p.Channel = ChannelManual
	return p
}

func manifestOwns(m Manifest, exe string) bool {
	for _, f := range m.Files {
		if strings.EqualFold(f, exe) || isWithin(f, exe) {
			return true
		}
	}
	return false
}

// packageManager recognizes paths owned by a package manager. norm is the
// lowercased, slash-separated executable path.
func packageManager(goos, norm string) string {
	switch {
	case strings.Contains(norm, "/node_modules/"):
		return "npm"
	case strings.Contains(norm, "/cellar/") || strings.Contains(norm, "/homebrew/") || strings.Contains(norm, "/linuxbrew/"):
		return "brew"
	case goos == "windows" && strings.Contains(norm, "/scoop/apps/"):
		return "scoop"
	case goos == "windows" && strings.Contains(norm, "/winget/packages/"):
		return "winget"
	case goos != "windows" && (strings.HasPrefix(norm, "/usr/bin/") || strings.HasPrefix(norm, "/bin/") ||
		strings.HasPrefix(norm, "/usr/sbin/") || strings.HasPrefix(norm, "/snap/") || strings.HasPrefix(norm, "/nix/")):
		return "system"
	}
	return ""
}

// ManagerCommand is the command that removes Kongtrol for a package manager
// ("" when it depends on the distribution).
func ManagerCommand(manager string) string {
	switch manager {
	case "brew":
		return "brew uninstall kongtrol"
	case "npm":
		return "npm uninstall -g kongtrol"
	case "scoop": // .goreleaser.yaml scoops
		return "scoop uninstall kongtrol"
	case "winget": // .goreleaser.yaml winget package_identifier
		return "winget uninstall Kongtrol.Kongtrol"
	}
	return ""
}
