package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate = validator.New()

// DefaultPaths returns candidate config file locations in priority order.
func DefaultPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".kongtrol", "kongtrol.yaml"),
		filepath.Join(home, ".kongtrol", "config.yaml"),
		"/etc/kongtrol/kongtrol.yaml",
		"kongtrol.yaml",
	}
}

// Load reads the config from path, applies environment overrides, and validates.
// If path is empty, the first existing file from DefaultPaths() is used.
func Load(path string) (*Config, error) {
	if path == "" {
		found := ""
		for _, candidate := range DefaultPaths() {
			if _, err := os.Stat(candidate); err == nil {
				found = candidate
				break
			}
		}
		if found == "" {
			return nil, fmt.Errorf("config: no config file found; run 'kongtrol init' to create one")
		}
		path = found
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	return Parse(data, path)
}

// Parse validates configuration contents before a caller replaces the file.
func Parse(data []byte, path string) (*Config, error) {
	// Expand environment variables in the YAML before parsing.
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}
	if err := validateSemantics(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	return &cfg, nil
}

// NewEmpty returns a config with no VPN profiles and every default applied.
// It is the first-run (setup mode) config the daemon serves before
// kongtrol.yaml exists. It does not pass Validate (vpns needs at least one
// profile), so it is never written as is: the first profile created from
// the dashboard makes it valid, and only then is the file saved.
func NewEmpty() *Config {
	cfg := &Config{VPNs: map[string]VPNConfig{}}
	applyDefaults(cfg)
	return cfg
}

// Validate runs the same struct-tag and semantic validation Load uses. It's
// exported for callers (e.g. the dashboard API) that mutate an already-loaded
// config in memory and need to check it before persisting.
func Validate(cfg *Config) error {
	if err := validate.Struct(cfg); err != nil {
		return fmt.Errorf("config: validation failed: %w", err)
	}
	return validateSemantics(cfg)
}

// applyDefaults fills in zero-value fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.Monitor.Dashboard.Port == 0 {
		cfg.Monitor.Dashboard.Port = 9741
	}
	if cfg.Monitor.Dashboard.Bind == "" {
		cfg.Monitor.Dashboard.Bind = "127.0.0.1"
	}
	if cfg.Monitor.HealthCheck.Interval == "" {
		cfg.Monitor.HealthCheck.Interval = "30s"
	}
	if cfg.Monitor.HealthCheck.Timeout == "" {
		cfg.Monitor.HealthCheck.Timeout = "10s"
	}
	if cfg.Monitor.History.FlushInterval == "" {
		cfg.Monitor.History.FlushInterval = "30s"
	}
	if cfg.Monitor.Scheduler.Interval == "" {
		cfg.Monitor.Scheduler.Interval = "1m"
	}
	if cfg.Monitor.SplitDNS.Interval == "" {
		cfg.Monitor.SplitDNS.Interval = "60s"
	}
	if cfg.Security.IntegrityCheck.Interval == "" {
		cfg.Security.IntegrityCheck.Interval = "120s"
	}
	if cfg.Security.AuditLog.MaxSizeMB == 0 {
		cfg.Security.AuditLog.MaxSizeMB = DefaultAuditMaxSizeMB
	}
	// 0 (unset, or written by an older version) means the default
	// (ADR-0064): shown and saved as the number really used.
	if cfg.Security.AuditLog.RetentionDays == 0 {
		cfg.Security.AuditLog.RetentionDays = DefaultAuditRetentionDays
	}
	if cfg.Security.AuditLog.MaxTotalMB == 0 {
		cfg.Security.AuditLog.MaxTotalMB = cfg.Security.AuditLog.EffectiveMaxTotalMB()
	}
	if cfg.Security.KillSwitch.Mode == "" {
		cfg.Security.KillSwitch.Mode = "strict"
	}
	// Expand ~ in file paths.
	home, _ := os.UserHomeDir()
	for name, vpn := range cfg.VPNs {
		vpn.Auth.Cert = expandHome(vpn.Auth.Cert, home)
		vpn.Auth.Key = expandHome(vpn.Auth.Key, home)
		vpn.ConfigFile = expandHome(vpn.ConfigFile, home)
		vpn.AppPath = expandHome(vpn.AppPath, home)
		vpn = vpn.WithDefaults()
		cfg.VPNs[name] = vpn
	}
	if cfg.Security.AuditLog.Path != "" {
		cfg.Security.AuditLog.Path = expandHome(cfg.Security.AuditLog.Path, home)
	}
	if cfg.Monitor.History.Path != "" {
		cfg.Monitor.History.Path = expandHome(cfg.Monitor.History.Path, home)
	}
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
