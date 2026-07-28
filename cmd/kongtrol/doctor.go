package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// doctorVersionCheckTimeout bounds each VPN client binary's --version-style
// probe. doctor is meant to be a quick, safe diagnostic — a misbehaving or
// hung client binary (or a PowerShell VersionInfo lookup that never
// returns) must not be able to make it hang indefinitely.
const doctorVersionCheckTimeout = 5 * time.Second

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your environment — binaries, certs, keychain, permissions",
	Long: `doctor validates your full Kongtrol stack without connecting to any VPN.
Run this before your first connection, or when diagnosing a teammate's setup.`,
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
		d := &doctor{}
		d.run()
		if outputJSON {
			if err := emitJSON(d.report()); err != nil {
				return err
			}
		} else {
			if d.warnings > 0 {
				fmt.Printf("\n%s\n", tuiWarn(cf("cli.doctor.summary.warn", d.warnings)))
			}
			if d.failures == 0 {
				fmt.Println("\n" + tuiOK(ct("cli.doctor.summary.ok")))
			}
		}
		if d.failures > 0 {
			return fmt.Errorf("%s", cf("cli.doctor.error.failed_checks", d.failures))
		}
		return nil
	},
}

var doctorFix bool

func init() {
	doctorCmd.Short = ct("cli.doctor.short")
	doctorCmd.Long = ct("cli.doctor.long")
	doctorCmd.Example = ct("cli.doctor.examples")
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
	return applied, nil
}

type doctorCheck struct {
	Section string `json:"section"`
	Label   string `json:"label"`
	Status  string `json:"status"` // ok|warn|fail
	Detail  string `json:"detail"`
}

type doctorReport struct {
	Checks   []doctorCheck `json:"checks"`
	Failures int           `json:"failures"`
	Warnings int           `json:"warnings"`
}

// ── doctor ────────────────────────────────────────────────────────────────────

type doctor struct {
	failures int
	warnings int
	sectionN string
	checks   []doctorCheck
}

func (d *doctor) run() {
	if !outputJSON {
		fmt.Println(stylePrompt.Render(sym("◎", "#")) + "  " + styleBright.Render(ct("cli.doctor.title")))
		fmt.Println(styleDim.Render(doctorRule()))
	}

	d.section(ct("cli.doctor.section.configuration"))
	_, cfg := d.checkConfig()

	d.section(ct("cli.doctor.section.binaries"))
	d.checkBinaries(cfg)

	d.section(ct("cli.doctor.section.certs"))
	d.checkCerts(cfg)

	d.section(ct("cli.doctor.section.keychain"))
	d.checkKeychain(cfg)

	d.section(ct("cli.doctor.section.permissions"))
	d.checkPermissions()

	d.section(ct("cli.doctor.section.runtime"))
	d.checkRuntimeNetworking(cfg)

	d.section(ct("cli.doctor.section.adapters_status"))
	d.checkAdapterStatus(cfg)

	d.section(ct("cli.doctor.section.adapters_registered"))
	d.checkAdapters(cfg)
}

func (d *doctor) report() doctorReport {
	return doctorReport{
		Checks:   d.checks,
		Failures: d.failures,
		Warnings: d.warnings,
	}
}

// ── config ────────────────────────────────────────────────────────────────────
