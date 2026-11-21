package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/doctor"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// The checks live in internal/doctor (shared with GET /api/v1/doctor and
// the dashboard); this command only renders them.
var doctorCmd = &cobra.Command{
	Use: "doctor",
	RunE: func(cmd *cobra.Command, args []string) error {
		if doctorFix {
			n, err := applyDoctorFixes()
			if err != nil {
				return err
			}
			if !outputJSON {
				fmt.Println(tuiInfo(cf("cli.doctor.fix.applied", n)))
			}
		}
		rep := runDoctor()
		if outputJSON {
			if err := emitJSON(rep); err != nil {
				return err
			}
		} else {
			if rep.Warnings > 0 {
				fmt.Printf("\n%s\n", tuiWarn(cf("cli.doctor.summary.warn", rep.Warnings)))
			}
			if rep.Failures == 0 {
				fmt.Println("\n" + tuiOK(ct("cli.doctor.summary.ok")))
			}
		}
		if rep.Failures > 0 {
			return fmt.Errorf("%s", cf("cli.doctor.error.failed_checks", rep.Failures))
		}
		return nil
	},
}

var doctorFix bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, ct("cli.doctor.flag.fix"))
	rootCmd.AddCommand(doctorCmd)
}

func applyDoctorFixes() (int, error) {
	applied := 0
	runDir := filepath.Join(filepath.Dir(pidFilePath()))
	if err := os.MkdirAll(runDir, 0o700); err == nil {
		applied++
	}
	paths := config.DefaultPaths()
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err == nil {
			applied++
		}
	}
	var cfgLoaded *config.Config
	if cfgPath != "" {
		if c, err := config.Load(cfgPath); err == nil {
			cfgLoaded = c
		}
	} else {
		for _, p := range paths {
			if c, err := config.Load(p); err == nil {
				cfgLoaded = c
				break
			}
		}
	}
	if cfgLoaded != nil && cfgLoaded.Security.AuditLog.Path != "" {
		if err := os.MkdirAll(filepath.Dir(cfgLoaded.Security.AuditLog.Path), 0o700); err == nil {
			applied++
		}
		todayAudit := security.DailyAuditLogPath(cfgLoaded.Security.AuditLog.Path, time.Now())
		if _, err := os.Stat(todayAudit); os.IsNotExist(err) {
			if err := os.WriteFile(todayAudit, []byte{}, 0o600); err == nil {
				applied++
			}
		}
	}
	// The helper does the network changes that need admin rights; --fix
	// installs or updates it like `kongtrol service install` (one OS
	// permission prompt). A refused or failed install is reported and the
	// checks below still run.
	if need := helperStateFn(); need != helperReady {
		if !outputJSON {
			fmt.Println(tuiInfo(ct("cli.doctor.fix.helper")))
		}
		if err := installHelperFn(need == helperOutdated); err != nil {
			if !outputJSON {
				fmt.Println(tuiWarn(err.Error()))
			}
		} else {
			applied++
		}
	}
	return applied, nil
}

// runDoctor runs the shared checks, printing each one as it arrives
// (unless --json), with the daemon's own state when one answers.
func runDoctor() doctor.Report {
	opt := doctor.Options{
		Lang:       cliLang(),
		ConfigPath: cfgPath,
		Version:    version,
		Daemon:     doctorDaemonState(),
	}
	if !outputJSON {
		fmt.Println(stylePrompt.Render(sym("◎", "#")) + "  " + styleBright.Render(ct("cli.doctor.title")))
		fmt.Println(styleDim.Render(doctorRule()))
		opt.OnSection = printDoctorSection
		opt.OnCheck = printDoctorCheck
	}
	return doctor.Run(opt)
}

// doctorDaemonState asks the daemon at the configured dashboard address
// for its tunnels and security state (nil when none answers). Read-only:
// the API token is read, never created.
func doctorDaemonState() *doctor.Daemon {
	c, err := config.Load(cfgPath)
	if err != nil {
		return nil
	}
	if apiToken == "" {
		tok, err := config.ReadAPIToken()
		if err != nil {
			return nil
		}
		apiToken = tok
	}
	tunnels, sec, err := fetchDaemonSnapshotAt(daemonctl.APIBase(c.Monitor.Dashboard.Bind, c.Monitor.Dashboard.Port))
	if err != nil {
		return nil
	}
	return doctorDaemonFromSnapshot(tunnels, sec)
}

func doctorDaemonFromSnapshot(tunnels []monitor.TunnelMetrics, sec *apiSecurityStatus) *doctor.Daemon {
	d := &doctor.Daemon{
		Tunnels:          make(map[string]vpn.Status, len(tunnels)),
		TunnelInterfaces: make(map[string]string, len(tunnels)),
	}
	for _, t := range tunnels {
		d.Tunnels[t.Name] = t.Status
		d.TunnelInterfaces[t.Name] = t.InterfaceName
	}
	if sec != nil {
		d.KillSwitchEnabled = sec.KillSwitchEnabled
		d.KillSwitchActive = sec.KillSwitch
		d.KillSwitch = sec.KillSwitchStatus
		d.DNSGuardActive = sec.DNSGuard
	}
	return d
}
