package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
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

// uninstallExtraFiles: a --config file and a custom audit log path, read
// from the config when it still loads.
func uninstallExtraFiles() []string {
	var extra []string
	if cfgPath != "" {
		extra = append(extra, cfgPath)
	}
	if c, err := config.Load(cfgPath); err == nil && c.Security.AuditLog.Path != "" {
		extra = append(extra, c.Security.AuditLog.Path)
	}
	return extra
}

func runUninstall(cmd *cobra.Command, _ []string) error {
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

	// 4. The app itself, per channel.
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

// confirmUninstall asks y/N (default No). With no terminal to ask on, the
// caller must pass --yes.
func confirmUninstall(purge bool) (bool, error) {
	if fi, err := os.Stdin.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false, errors.New(ct("cli.uninstall.needs_yes"))
	}
	key := "cli.uninstall.confirm"
	if purge {
		key = "cli.uninstall.confirm_purge"
	}
	fmt.Print(ct(key))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "s", "si", "sí", "y", "yes":
		return true, nil
	}
	return false, nil
}

// removeScriptInstall deletes what install.sh / install.ps1 listed, then
// its manifest. The running executable is removed last (on Windows after
// this process exits).
func removeScriptInstall(p uninstall.Plan) {
	for _, a := range p.Autostart {
		removeListed(a)
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

func removeListed(path string) {
	if _, err := os.Lstat(path); err != nil {
		return
	}
	if err := os.RemoveAll(path); err != nil {
		say(tuiWarn(cf("cli.uninstall.remove_failed", path)))
		return
	}
	say(tuiOK(cf("cli.uninstall.removed", path)))
}

// errNotInPath: the directory was not in the user PATH.
var errNotInPath = errors.New("not in PATH")
