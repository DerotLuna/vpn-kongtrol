package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/autostart"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/uninstall"
)

// `kongtrol uninstall` (ADR-0034). Keeping data is the default; --purge also
// deletes ~/.kongtrol and Kongtrol's keychain entries; --data-only deletes
// only the data (the Windows uninstaller and the dashboard use the same
// purge). Files are deleted only when Kongtrol's own install scripts put
// them there (install.manifest); package managers keep ownership of theirs.

var (
	uninstallPurge    bool
	uninstallDataOnly bool
	uninstallYes      bool
	uninstallDryRun   bool
)

var uninstallCmd = &cobra.Command{
	Use:  "uninstall",
	Args: cobra.NoArgs,
	// No config load: it must work with a broken or missing kongtrol.yaml.
	Annotations: map[string]string{annotationPrefsOnly: "true"},
	RunE:        runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, ct("cli.flag.uninstall.purge"))
	uninstallCmd.Flags().BoolVar(&uninstallDataOnly, "data-only", false, ct("cli.flag.uninstall.data_only"))
	uninstallCmd.Flags().BoolVarP(&uninstallYes, "yes", "y", false, ct("cli.flag.uninstall.yes"))
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, ct("cli.flag.uninstall.dry_run"))
	rootCmd.AddCommand(uninstallCmd)
}

// uninstallEnv describes this copy of Kongtrol for uninstall.Detect.
func uninstallEnv() uninstall.Env {
	exe, _ := currentExecutable()
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return uninstall.Env{
		GOOS:         runtime.GOOS,
		Exe:          exe,
		DataDir:      kongtrolDataDir(),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
	}
}

// uninstallExtraFiles: a --config file and the daily files of a custom
// audit log path (security.AuditLogFiles), read
// from the config when it still loads.
func uninstallExtraFiles() []string {
	var extra []string
	if cfgPath != "" {
		extra = append(extra, cfgPath)
	}
	if c, err := config.Load(cfgPath); err == nil && c.Security.AuditLog.Path != "" {
		extra = append(extra, security.AuditLogFiles(c.Security.AuditLog.Path)...)
	}
	return extra
}

func runUninstall(cmd *cobra.Command, _ []string) error {
	if runViaSudo(os.Geteuid(), os.Getenv) {
		// Under sudo, doas or pkexec, HOME may be root's or the user's
		// depending on the OS and sudoers (env_keep). Resolving the real
		// user (SUDO_UID + getpwuid) is not enough: root would still stop
		// every user's tray, miss the user's daemon (listener-owner check)
		// and keychain, and delete inside a folder the user controls.
		// Refuse instead of acting on the wrong user (ADR-0100).
		say(tuiWarn(ct("cli.uninstall.no_sudo")))
		say(tuiInfo(ct("cli.uninstall.no_sudo.fix")))
		return alreadyReported(errors.New("uninstall run via sudo"))
	}
	plan := uninstall.Detect(uninstallEnv(), uninstallPurge, uninstallDataOnly)
	extra := uninstallExtraFiles()
	profiles := uninstall.ProfileNames(uninstall.ConfigFiles(plan.DataDir, extra...))
	if outputJSON && uninstallDryRun {
		return emitJSON(map[string]any{"plan": plan, "profiles": profiles})
	}
	printUninstallPlan(plan, len(profiles))
	if uninstallDryRun {
		say(tuiInfo(ct("cli.uninstall.dry_run")))
		return nil
	}
	if !uninstallYes {
		ok, err := confirmUninstall(plan.Purge)
		if err != nil {
			return err
		}
		if !ok {
			say(tuiInfo(ct("cli.uninstall.cancelled")))
			return nil
		}
	}

	// 1. Tunnels down and Kongtrol closed: the daemon disconnects every
	// tunnel, lifts kill switch blocks and restores DNS on shutdown.
	stopTray()
	stopDaemon()
	say(tuiOK(ct("cli.uninstall.stopped")))

	// 2. Network helper (the Windows uninstaller removes it itself).
	if !plan.DataOnly && plan.Channel != uninstall.ChannelWindowsInstaller && privhelper.QueryStatus().Installed {
		if err := runServiceUninstall(cmd, nil); err != nil {
			say(tuiWarn(ct("cli.uninstall.helper_failed")))
		}
	}

	// 3. Data and saved passwords, only when asked.
	if plan.Purge {
		rep := uninstall.Purge(uninstall.PurgeOptions{DataDir: plan.DataDir, ExtraFiles: extra, KeepManifest: plan.DataOnly})
		if err := rep.Err(); err != nil {
			say(tuiErr(cf("cli.uninstall.purge_failed", err)))
			say(tuiInfo(ct("cli.uninstall.purge_failed_fix")))
			return alreadyReported(err)
		}
		say(tuiOK(cf("cli.uninstall.purged", len(rep.Credentials))))
	}
	if plan.DataOnly {
		say(tuiInfo(ct("cli.uninstall.data_only_done")))
		return nil
	}

	// 4. A sign-in start that Kongtrol's own setting wrote for this tray
	// (ADR-0098); the Windows installer removes its entry itself.
	if plan.Channel != uninstall.ChannelWindowsInstaller {
		if tray, ok := autostart.TrayPath(); ok {
			_ = autostart.Remove(tray)
		}
	}

	// 5. The app itself, per channel.
	switch plan.Channel {
	case uninstall.ChannelScript:
		removeScriptInstall(plan)
	case uninstall.ChannelWindowsInstaller:
		if err := startDetached(plan.Uninstaller); err != nil {
			say(tuiWarn(ct("cli.uninstall.windows_failed")))
		} else {
			say(tuiInfo(ct("cli.uninstall.windows_next")))
		}
	case uninstall.ChannelPackageManager:
		if c := uninstall.ManagerCommand(plan.Manager); c != "" {
			say(tuiInfo(cf("cli.uninstall.pkg_next", c)))
		} else {
			say(tuiInfo(ct("cli.uninstall.pkg_next_generic")))
		}
	default:
		say(tuiInfo(cf("cli.uninstall.manual_next", plan.Exe)))
	}
	if plan.Channel == uninstall.ChannelScript {
		say(tuiOK(ct("cli.uninstall.done")))
	}
	if !plan.Purge {
		say(tuiInfo(cf("cli.uninstall.done_kept", plan.DataDir)))
	}
	return nil
}

func printUninstallPlan(p uninstall.Plan, profiles int) {
	say(tuiInfo(ct("cli.uninstall.plan.title")))
	step := func(s string) { say("    · " + s) }
	step(ct("cli.uninstall.step.stop"))
	if !p.DataOnly {
		if p.Channel != uninstall.ChannelWindowsInstaller && privhelper.QueryStatus().Installed {
			step(ct("cli.uninstall.step.helper"))
		}
		switch p.Channel {
		case uninstall.ChannelScript:
			for _, a := range p.Autostart {
				step(cf("cli.uninstall.step.autostart", a))
			}
			if len(p.RunValues) > 0 {
				step(cf("cli.uninstall.step.autostart", runValueLabel))
			}
			for _, d := range p.PathEntries {
				step(cf("cli.uninstall.step.path", d))
			}
			for _, f := range p.Files {
				step(cf("cli.uninstall.step.file", f))
			}
		case uninstall.ChannelWindowsInstaller:
			step(ct("cli.uninstall.step.windows"))
		case uninstall.ChannelPackageManager:
			step(cf("cli.uninstall.step.pkg", p.Manager))
		default:
			step(cf("cli.uninstall.step.manual", p.Exe))
		}
	}
	if p.Purge {
		step(cf("cli.uninstall.step.purge", p.DataDir, profiles))
	} else {
		step(cf("cli.uninstall.step.keep", p.DataDir))
	}
}

// confirmUninstall asks y/N (default No) through the shared prompt
// (prompt.go). With no terminal to ask on (or --json/-q), the caller must
// pass --yes.
func confirmUninstall(purge bool) (bool, error) {
	if !canPrompt() {
		return false, errors.New(ct("cli.uninstall.needs_yes"))
	}
	key := "cli.uninstall.confirm"
	if purge {
		key = "cli.uninstall.confirm_purge"
	}
	return askYesNo(ct(key), false), nil
}

// removeScriptInstall deletes what install.sh / install.ps1 listed, then
// its manifest. The running executable is removed last (on Windows after
// this process exits).
func removeScriptInstall(p uninstall.Plan) {
	for _, a := range p.Autostart {
		removeListed(a)
	}
	for _, tray := range p.RunValues {
		if err := removeRunValue(tray); err == nil {
			say(tuiOK(cf("cli.uninstall.removed", runValueLabel)))
		}
	}
	for _, d := range p.PathEntries {
		if err := removeUserPathEntry(d); err == nil {
			say(tuiOK(cf("cli.uninstall.removed", d+" (PATH)")))
		}
	}
	var self string
	for _, f := range p.Files {
		if strings.EqualFold(filepath.Clean(f), filepath.Clean(p.Exe)) && runtime.GOOS == "windows" {
			self = f
			continue
		}
		removeListed(f)
	}
	_ = os.Remove(filepath.Join(p.DataDir, uninstall.ManifestName))
	if self != "" {
		if err := deleteAfterExit(self); err != nil {
			say(tuiWarn(cf("cli.uninstall.remove_failed", self)))
		} else {
			say(tuiOK(cf("cli.uninstall.removed", self)))
		}
	}
}

// runViaSudo reports a root process started for another user by sudo
// (SUDO_USER / SUDO_UID, sudo(8)), doas (DOAS_USER, doas(1)) or pkexec
// (PKEXEC_UID, pkexec(1)). euid is -1 on Windows. A root shell of its own
// (su -, a root login) is not refused: its HOME is root's own.
func runViaSudo(euid int, getenv func(string) string) bool {
	if euid != 0 {
		return false
	}
	otherName := func(k string) bool { v := getenv(k); return v != "" && v != "root" }
	otherUID := func(k string) bool { v := getenv(k); return v != "" && v != "0" }
	return otherName("SUDO_USER") || otherUID("SUDO_UID") || otherName("DOAS_USER") || otherUID("PKEXEC_UID")
}

// removeListed deletes one manifest entry (already filtered to Kongtrol's
// own names by uninstall.ReadManifest). Only Kongtrol.app is a folder and
// removed recursively; anything else must be a file or link.
func removeListed(path string) {
	fi, err := os.Lstat(path)
	if err != nil {
		return
	}
	remove := os.Remove
	if fi.IsDir() {
		if filepath.Base(path) != uninstall.AppBundleName {
			say(tuiWarn(cf("cli.uninstall.remove_failed", path)))
			return
		}
		remove = os.RemoveAll
	}
	if err := remove(path); err != nil {
		say(tuiWarn(cf("cli.uninstall.remove_failed", path)))
		return
	}
	say(tuiOK(cf("cli.uninstall.removed", path)))
}

// errNotInPath: the directory was not in the user PATH.
var errNotInPath = errors.New("not in PATH")

// runValueLabel names the per-user autostart value install.ps1 writes (the
// same value the Windows installer uses, ADR-0075).
const runValueLabel = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\Kongtrol`

// errRunValueNotOurs: the Run value is missing or points to another copy.
var errRunValueNotOurs = errors.New("run value missing or not ours")

// runValueIsFor reports whether a Run value's data starts the given tray
// executable: `"C:\...\kongtrol-tray.exe"` or the bare path, any case.
func runValueIsFor(data, tray string) bool {
	d := strings.TrimSpace(data)
	if strings.HasPrefix(d, `"`) {
		if end := strings.Index(d[1:], `"`); end >= 0 {
			d = d[1 : end+1]
		}
	}
	return d != "" && strings.EqualFold(filepath.Clean(d), filepath.Clean(tray))
}
