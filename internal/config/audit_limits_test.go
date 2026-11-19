package config_test

import (
	"errors"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

const auditProfileYAML = `
vpns:
  office:
    type: wireguard
    config: /tmp/office.conf
    auth:
      method: credentials
security:
  audit_log:
    path: /tmp/audit.log
`

// The logger's own defaults (used without a config) match the config's.
func TestAuditDefaultsMatchLogger(t *testing.T) {
	if config.DefaultAuditRetentionDays != security.DefaultAuditRetentionDays ||
		config.MaxAuditRetentionDays != security.MaxAuditRetentionDays ||
		config.MaxAuditMaxSizeMB != security.MaxAuditFileSizeMB ||
		config.DefaultAuditMaxTotalMB != security.DefaultAuditMaxTotalMB ||
		config.MaxAuditMaxTotalMB != security.MaxAuditTotalMB {
		t.Fatal("internal/config and internal/security audit limits differ")
	}
}

// Migration (ADR-0064): retention_days 0 or unset loads as 30 days and the
// total cap as 500 MB, or one file's size when that is larger.
func TestAuditLimits_UnsetOrZeroLoadsDefaults(t *testing.T) {
	for name, extra := range map[string]string{
		"unset":   "",
		"zero":    "    retention_days: 0\n    max_total_mb: 0\n",
		"bigfile": "    max_size_mb: 1024\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.Load(writeTempConfig(t, auditProfileYAML+extra))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			al := cfg.Security.AuditLog
			if al.RetentionDays != 30 {
				t.Errorf("retention_days = %d, want 30", al.RetentionDays)
			}
			wantTotal := 500
			if name == "bigfile" {
				wantTotal = 1024
			}
			if al.MaxTotalMB != wantTotal {
				t.Errorf("max_total_mb = %d, want %d", al.MaxTotalMB, wantTotal)
			}
		})
	}
}

func TestAuditLimits_Validation(t *testing.T) {
	cases := []struct {
		name    string
		al      config.AuditLogConfig
		wantKey string
	}{
		{"zero means defaults", config.AuditLogConfig{}, ""},
		{"one day", config.AuditLogConfig{RetentionDays: 1, MaxSizeMB: 10, MaxTotalMB: 10}, ""},
		{"retention too high", config.AuditLogConfig{RetentionDays: 366}, "config.err.audit_retention"},
		{"retention negative", config.AuditLogConfig{RetentionDays: -1}, "config.err.audit_retention"},
		{"total too high", config.AuditLogConfig{MaxTotalMB: 10241}, "config.err.audit_max_total"},
		{"total below one file", config.AuditLogConfig{MaxSizeMB: 200, MaxTotalMB: 100}, "config.err.audit_total_small"},
		{"total below default file", config.AuditLogConfig{MaxTotalMB: 50}, "config.err.audit_total_small"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{VPNs: map[string]config.VPNConfig{
				"office": {Type: "wireguard", ConfigFile: "/tmp/office.conf", Auth: config.AuthConfig{Method: "credentials"}},
			}}
			cfg.Security.AuditLog = tc.al
			err := config.Validate(cfg)
			if tc.wantKey == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *config.ValidationError
			if !errors.As(err, &ve) || ve.Key != tc.wantKey {
				t.Fatalf("got %v, want %s", err, tc.wantKey)
			}
		})
	}
}

func TestAuditLimits_Effective(t *testing.T) {
	al := config.AuditLogConfig{}
	if al.EffectiveRetentionDays() != 30 || al.EffectiveMaxSizeMB() != 100 || al.EffectiveMaxTotalMB() != 500 {
		t.Fatalf("defaults: %d days, %d MB/file, %d MB total", al.EffectiveRetentionDays(), al.EffectiveMaxSizeMB(), al.EffectiveMaxTotalMB())
	}
	al = config.AuditLogConfig{RetentionDays: 7, MaxSizeMB: 800, MaxTotalMB: 0}
	if al.EffectiveRetentionDays() != 7 || al.EffectiveMaxTotalMB() != 800 {
		t.Fatalf("set: %d days, %d MB total", al.EffectiveRetentionDays(), al.EffectiveMaxTotalMB())
	}
}
