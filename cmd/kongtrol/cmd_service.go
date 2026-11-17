package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// Hidden flags passed from the unprivileged `service install|uninstall` to
// its own elevated re-run (UAC / sudo). The identity flags record who
// installed the helper: an elevated process may run as a different
// account (over-the-shoulder admin), so the unprivileged parent sends it.
var (
	svcElevatedStep bool
	svcUserSID      string
	svcUserName     string
	svcUID          int
	svcGID          int
)

// serviceCmd groups the privileged helper commands. Its own pre-run
// replaces the root one: no config load is needed to (un)install.
var serviceCmd = &cobra.Command{
	Use:  "service",
	Args: cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputPlain {
			_ = os.Setenv("NO_COLOR", "1")
		}
		if !svcElevatedStep && !outputJSON && !outputQuiet {
			PrintHeader(version)
		}
		return nil
	},
}

var serviceInstallCmd = &cobra.Command{Use: "install", Args: cobra.NoArgs, RunE: runServiceInstall}
var serviceUninstallCmd = &cobra.Command{Use: "uninstall", Args: cobra.NoArgs, RunE: runServiceUninstall}
var serviceStatusCmd = &cobra.Command{Use: "status", Args: cobra.NoArgs, RunE: runServiceStatus}

// helperCmd is the hidden entrypoint the Windows service / launchd daemon /
// systemd unit runs. Users never call it.
var helperCmd = &cobra.Command{
	Use:               "helper",
	Hidden:            true,
	Args:              cobra.NoArgs,
	PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
}

var helperRunCmd = &cobra.Command{
	Use:  "run",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return privhelper.Run(contextWithSignal())
	},
}

func init() {
	privops.BuildVersion = version
	helperCmd.Short = ct("cli.helper.short")
	helperRunCmd.Short = ct("cli.helper.short")

	for _, c := range []*cobra.Command{serviceInstallCmd, serviceUninstallCmd} {
		f := c.Flags()
		f.BoolVar(&svcElevatedStep, "elevated-step", false, "")
		f.StringVar(&svcUserSID, "user-sid", "", "")
		f.StringVar(&svcUserName, "user-name", "", "")
		f.IntVar(&svcUID, "uid", -1, "")
		f.IntVar(&svcGID, "gid", -1, "")
		for _, n := range []string{"elevated-step", "user-sid", "user-name", "uid", "gid"} {
			_ = f.MarkHidden(n)
		}
	}
	serviceCmd.AddCommand(serviceInstallCmd, serviceUninstallCmd, serviceStatusCmd)
	helperCmd.AddCommand(helperRunCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(helperCmd)
}

func say(line string) {
	if !outputQuiet && !outputJSON {
		fmt.Println(line)
	}
}

// guidedError is a translated failure line plus one "what to do" line.
func guidedError(msgKey string, cause error, hintKey string) error {
	msg := ct(msgKey)
	if cause != nil {
		msg = cf(msgKey+"_detail", cause)
	}
	return fmt.Errorf("%s\n   %s", msg, ct(hintKey))
}

func currentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func runServiceInstall(cmd *cobra.Command, args []string) error {
	if svcElevatedStep {
		exe, err := currentExecutable()
		if err != nil {
			return err
		}
		opts, err := elevatedInstallOptions()
		if err != nil {
			return err
		}
		opts.SourceExe, opts.Version = exe, version
		return privhelper.Install(opts)
	}

	say(tuiInfo(ct("service.install.start")))
	if privops.IsElevated() {
		exe, err := currentExecutable()
		if err != nil {
			return guidedError("service.install.failed", err, "service.install.failed_hint")
		}
		opts, err := elevatedInstallOptions()
		if err != nil {
			return guidedError("service.install.failed", err, "service.install.failed_hint")
		}
		opts.SourceExe, opts.Version = exe, version
		if err := privhelper.Install(opts); err != nil {
			return guidedError("service.install.failed", err, "service.install.failed_hint")
		}
	} else {
		say(tuiInfo(ct(elevationPromptKey())))
		if err := runElevatedStep("install", identityArgs()); err != nil {
			if errors.Is(err, privhelper.ErrElevationCancelled) {
				return guidedError("service.elevation.cancelled", nil, "service.elevation.cancelled_hint")
			}
			return guidedError("service.install.failed", err, "service.install.failed_hint")
		}
	}

	say(tuiInfo(ct("service.install.verifying")))
	res, err := waitForHelper(15 * time.Second)
	if err != nil {
		return guidedError("service.install.no_response", nil, "service.install.no_response_hint")
	}
	if outputJSON {
		return emitJSON(map[string]any{"installed": true, "running": true, "helper_version": res.Version})
	}
	say(tuiOK(ct("service.install.ok")))
	say(tuiInfo(ct("service.install.next")))
	return nil
}

func waitForHelper(d time.Duration) (privops.PingResult, error) {
	c := privops.NewClient()
	deadline := time.Now().Add(d)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		res, err := c.Ping(ctx)
		cancel()
		if err == nil {
			return res, nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return privops.PingResult{}, lastErr
}

func runServiceUninstall(cmd *cobra.Command, args []string) error {
	if svcElevatedStep {
		return privhelper.Uninstall()
	}
	say(tuiInfo(ct("service.uninstall.start")))
	var err error
	if privops.IsElevated() {
		err = privhelper.Uninstall()
	} else {
		say(tuiInfo(ct(elevationPromptKey())))
		err = runElevatedStep("uninstall", nil)
	}
	if err != nil {
		if errors.Is(err, privhelper.ErrElevationCancelled) {
			return guidedError("service.elevation.cancelled", nil, "service.elevation.cancelled_hint")
		}
		return guidedError("service.uninstall.failed", err, "service.uninstall.failed_hint")
	}
	if outputJSON {
		return emitJSON(map[string]any{"installed": false})
	}
	say(tuiOK(ct("service.uninstall.ok")))
	say(tuiInfo(ct("service.uninstall.next")))
	return nil
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	st := privhelper.QueryStatus()
	report := map[string]any{"installed": st.Installed, "running": st.Running, "cli_version": version}
	defer func() {
		if outputJSON {
			_ = emitJSON(report)
		}
	}()
	if !st.Installed {
		say(tuiWarn(ct("service.status.not_installed")))
		say(tuiInfo(ct("service.error.not_installed")))
		return nil
	}
	say(tuiOK(cf("service.status.installed", st.BinaryPath)))
	toolIssues := reportServiceTools(report)
	if !st.Running {
		say(tuiWarn(ct("service.status.stopped")))
		say(tuiInfo(ct("service.status.stopped_hint")))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := privops.NewClient().Ping(ctx)
	switch {
	case errors.Is(err, privops.ErrDenied):
		report["error"] = "denied"
		say(tuiErr(ct("service.status.denied")))
		say(tuiInfo(ct("service.status.denied_hint")))
		return nil
	case err != nil:
		report["error"] = "no_response"
		say(tuiErr(ct("service.install.no_response")))
		say(tuiInfo(ct("service.install.no_response_hint")))
		return nil
	}
	report["helper_version"] = res.Version
	say(tuiOK(cf("service.status.running", res.Version)))
	if res.Version != version || res.Protocol != privops.ProtocolVersion {
		report["version_mismatch"] = true
		say(tuiWarn(cf("service.status.version_mismatch", res.Version, version)))
		say(tuiInfo(ct("service.status.version_mismatch_hint")))
		return nil
	}
	if !toolIssues {
		say(tuiOK(ct("service.status.ok")))
	}
	return nil
}

// reportServiceTools warns about Homebrew tool copies whose source changed
// since install, and tools install could not use. It reports whether any
// warning was shown.
func reportServiceTools(report map[string]any) bool {
	changed, unusable, err := privhelper.CheckTools()
	if err != nil {
		return false // an unreadable manifest is not worth alarming about here
	}
	if len(changed) > 0 {
		names := make([]string, 0, len(changed))
		for _, c := range changed {
			names = append(names, c.Tool)
			say(tuiWarn(cf("service.status.tool_changed", c.Tool, c.Source)))
		}
		say(tuiInfo(ct("service.status.tool_changed_hint")))
		report["tools_changed"] = names
	}
	if len(unusable) > 0 {
		names := make([]string, 0, len(unusable))
		for _, u := range unusable {
			names = append(names, u.Tool)
			say(tuiWarn(cf("service.status.tool_unusable", u.Tool, u.Source, u.Reason)))
		}
		say(tuiInfo(ct("service.status.tool_unusable_hint")))
		report["tools_unusable"] = names
	}
	return len(changed)+len(unusable) > 0
}
