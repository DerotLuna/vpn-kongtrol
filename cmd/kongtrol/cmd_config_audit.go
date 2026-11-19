package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// config audit-log: the dashboard's "Ajustes del registro" (Actividad) in
// the CLI (CLI-GUI parity, ADR-0064). Without flags it shows how long the
// audit log is kept and how much space it may use; with flags it changes
// that through the daemon (applied at once) or, with no daemon, in
// kongtrol.yaml (comments kept), after the same validation.
var (
	auditDaysFlag     int
	auditMaxSizeFlag  int
	auditMaxTotalFlag int
)

var configAuditLogCmd = &cobra.Command{
	Use:  "audit-log",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var patch daemonctl.AuditLogPatch
		if cmd.Flags().Changed("days") {
			patch.RetentionDays = &auditDaysFlag
		}
		if cmd.Flags().Changed("max-size-mb") {
			patch.MaxSizeMB = &auditMaxSizeFlag
		}
		if cmd.Flags().Changed("max-total-mb") {
			patch.MaxTotalMB = &auditMaxTotalFlag
		}
		changing := patch.RetentionDays != nil || patch.MaxSizeMB != nil || patch.MaxTotalMB != nil
		base := daemonAPIBase()
		running := probeDaemonAPI(base)
		if changing {
			if running {
				if err := daemonClient(base).SetAuditLog(patch); err != nil {
					return err
				}
			} else if err := setAuditLogInFile(activeCfgPath, patch); err != nil {
				return err
			}
		}
		st := auditLogFromConfig()
		if running {
			if got, err := daemonClient(base).AuditLog(); err == nil {
				st = got
			}
		}
		return printAuditLog(st, changing, running)
	},
}

// auditLogFromConfig is the loaded kongtrol.yaml's audit log, as in use.
func auditLogFromConfig() daemonctl.AuditLogSettings {
	var al config.AuditLogConfig
	if cfg != nil {
		al = cfg.Security.AuditLog
	}
	return daemonctl.AuditLogSettings{
		Path: al.Path, Sign: al.Sign,
		MaxSizeMB: al.EffectiveMaxSizeMB(), RetentionDays: al.EffectiveRetentionDays(), MaxTotalMB: al.EffectiveMaxTotalMB(),
	}
}

func printAuditLog(st daemonctl.AuditLogSettings, changed, running bool) error {
	var used int64
	files := 0
	if st.Path != "" {
		for _, p := range security.AuditLogFiles(st.Path) {
			if fi, err := os.Stat(p); err == nil {
				used += fi.Size()
				files++
			}
		}
	}
	if outputJSON {
		return emitJSON(map[string]any{
			"enabled": st.Path != "", "path": st.Path, "retention_days": st.RetentionDays,
			"max_size_mb": st.MaxSizeMB, "max_total_mb": st.MaxTotalMB, "sign": st.Sign,
			"files": files, "used_bytes": used, "applied_now": running,
		})
	}
	if changed {
		fmt.Println(tuiOK(ct("cli.audit_log.saved")))
		if !running {
			fmt.Println(tuiInfo(paint(cDim, ct("cli.config.toggle.saved_file"))))
		}
	}
	if st.Path == "" {
		fmt.Println(tuiInfo(ct("cli.audit_log.off")))
	} else {
		fmt.Println(tuiInfo(cf("cli.audit_log.path", st.Path)))
	}
	fmt.Println(tuiInfo(cf("cli.audit_log.limits", st.RetentionDays, st.MaxSizeMB, st.MaxTotalMB)))
	if st.Path != "" {
		fmt.Println(tuiInfo(cf("cli.audit_log.used", files, float64(used)/(1<<20))))
	}
	fmt.Println(tuiInfo(paint(cDim, ct("cli.audit_log.next"))))
	return nil
}

// setAuditLogInFile writes the set fields under security.audit_log after
// checking them like the daemon does (config.Validate on a copy).
func setAuditLogInFile(path string, p daemonctl.AuditLogPatch) error {
	var trial config.Config
	if cfg != nil {
		trial = *cfg
	}
	al := &trial.Security.AuditLog
	if p.RetentionDays != nil {
		al.RetentionDays = *p.RetentionDays
	}
	if p.MaxSizeMB != nil {
		al.MaxSizeMB = *p.MaxSizeMB
	}
	if p.MaxTotalMB != nil {
		al.MaxTotalMB = *p.MaxTotalMB
	}
	if err := config.Validate(&trial); err != nil {
		return localizeConfigErr(err)
	}
	return editConfigYAML(path, func(root *yaml.Node) {
		node := childMapping(childMapping(root, "security"), "audit_log")
		for _, kv := range []struct {
			key string
			v   *int
		}{{"max_size_mb", p.MaxSizeMB}, {"retention_days", p.RetentionDays}, {"max_total_mb", p.MaxTotalMB}} {
			if kv.v != nil {
				setMapping(node, kv.key, intNode(strconv.Itoa(*kv.v)))
			}
		}
	})
}

func init() {
	configAuditLogCmd.Flags().IntVar(&auditDaysFlag, "days", 0, ct("cli.audit_log.flag.days"))
	configAuditLogCmd.Flags().IntVar(&auditMaxSizeFlag, "max-size-mb", 0, ct("cli.audit_log.flag.max_size"))
	configAuditLogCmd.Flags().IntVar(&auditMaxTotalFlag, "max-total-mb", 0, ct("cli.audit_log.flag.max_total"))
	configCmd.AddCommand(configAuditLogCmd)
}
