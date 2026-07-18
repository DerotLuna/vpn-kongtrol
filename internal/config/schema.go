package config

// Config is the root configuration structure for VPN Kongtrol.
type Config struct {
	VPNs     map[string]VPNConfig   `yaml:"vpns"     validate:"required,min=1,dive"`
	Groups   map[string]GroupConfig `yaml:"groups"`
	Policies []PolicyRule           `yaml:"policies"`
	Security SecurityConfig         `yaml:"security"`
	Monitor  MonitorConfig          `yaml:"monitor"`
}

// GroupConfig defines a named set of VPN profiles that can be
// connected / disconnected together with --group <name>.
type GroupConfig struct {
	Profiles []string `yaml:"profiles" validate:"required,min=1"`
}

// VPNConfig holds the configuration for a single VPN profile.
type VPNConfig struct {
	Type       string `yaml:"type"        validate:"required,oneof=forticlient openvpn protonvpn ciscoanyconnect wireguard globalprotect tailscale cloudflarewarp"`
	Version    string `yaml:"version"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	TunnelName string `yaml:"tunnel_name"`
	// FortiClient-only: when true, GUI automation accepts certificate warnings.
	// Keep false in production to avoid MITM risk.
	AllowInsecureCert bool       `yaml:"allow_insecure_cert"`
	ConfigFile        string     `yaml:"config"`      // .ovpn / wg .conf file path
	Server            string     `yaml:"server"`      // server identifier (protonvpn)
	Protocol          string     `yaml:"protocol"`    // wireguard|openvpn (protonvpn)
	BinaryPath        string     `yaml:"binary_path"` // override adapter binary location (optional)
	Auth              AuthConfig `yaml:"auth"`
	Priority          int        `yaml:"priority"`
	KillSwitch        *bool      `yaml:"kill_switch"` // optional per-profile override
}

// AuthConfig defines authentication credentials for a VPN profile.
// Passwords are never stored here — they live in the OS keychain.
type AuthConfig struct {
	// certificate | credentials | certificate+credentials
	Method           string `yaml:"method" validate:"required"`
	Cert             string `yaml:"cert"`              // path to client certificate
	Key              string `yaml:"key"`               // path to private key
	Username         string `yaml:"username"`          // plaintext username (low-sensitivity)
	UsernameKeychain string `yaml:"username_keychain"` // keychain key name for username
	PasswordKeychain string `yaml:"password_keychain"` // keychain key name for password
}

// PolicyRule maps traffic to a specific VPN profile.
type PolicyRule struct {
	Name  string    `yaml:"name" validate:"required"`
	Match MatchSpec `yaml:"match"`
	Via   string    `yaml:"via"  validate:"required"`
}

// MatchSpec defines which traffic a policy rule matches.
type MatchSpec struct {
	IPRanges []string `yaml:"ip_ranges"` // CIDR notation
	Domains  []string `yaml:"domains"`   // glob patterns (*.example.com)
	Apps     []string `yaml:"apps"`      // executable names or glob patterns (experimental)
}

// SecurityConfig holds all security-related settings.
type SecurityConfig struct {
	KillSwitch     KillSwitchConfig     `yaml:"kill_switch"`
	DNSGuard       DNSGuardConfig       `yaml:"dns_guard"`
	IntegrityCheck IntegrityCheckConfig `yaml:"integrity_check"`
	LeakDetection  LeakDetectionConfig  `yaml:"leak_detection"`
	AuditLog       AuditLogConfig       `yaml:"audit_log"`
}

// KillSwitchConfig configures the kill switch behavior.
type KillSwitchConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Mode     string `yaml:"mode" validate:"omitempty,oneof=strict loose"` // strict blocks all; loose allows LAN
	AllowLAN bool   `yaml:"allow_lan"`
}

// DNSGuardConfig configures DNS leak prevention.
type DNSGuardConfig struct {
	Enabled     bool   `yaml:"enabled"`
	FallbackDNS string `yaml:"fallback_dns"`
}

// IntegrityCheckConfig configures tunnel integrity verification.
type IntegrityCheckConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Interval    string            `yaml:"interval"`     // e.g. "120s"
	ExpectedIPs map[string]string `yaml:"expected_ips"` // vpn_name → expected public IP
}

// LeakDetectionConfig configures continuous IP/DNS leak detection.
type LeakDetectionConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"` // e.g. "60s"
	// notify | killswitch_and_notify
	Action string `yaml:"action" validate:"omitempty,oneof=notify killswitch_and_notify"`
}

// AuditLogConfig configures the tamper-evident audit log.
type AuditLogConfig struct {
	Path      string `yaml:"path"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	Sign      bool   `yaml:"sign"` // HMAC-SHA256 per entry
}

// MonitorConfig holds monitoring and dashboard settings.
type MonitorConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Dashboard   DashboardConfig   `yaml:"dashboard"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	History     HistoryConfig     `yaml:"history"`
	Scheduler   SchedulerConfig   `yaml:"scheduler"`
	SplitDNS    SplitDNSConfig    `yaml:"split_dns"`
	Alerts      AlertsConfig      `yaml:"alerts"`
}

// DashboardConfig configures the embedded web dashboard.
type DashboardConfig struct {
	Port int    `yaml:"port"` // default 9741
	Bind string `yaml:"bind"` // default 127.0.0.1
}

// HealthCheckConfig configures per-tunnel health checks.
type HealthCheckConfig struct {
	Interval string `yaml:"interval"` // e.g. "30s"
	Timeout  string `yaml:"timeout"`  // e.g. "10s"
}

// HistoryConfig stores rolling profile stability metrics.
type HistoryConfig struct {
	Path          string `yaml:"path"`           // e.g. ~/.kongtrol/history.json
	FlushInterval string `yaml:"flush_interval"` // e.g. "30s"
}

// SchedulerConfig activates profile groups by schedule windows.
type SchedulerConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Interval string         `yaml:"interval"` // e.g. "1m"
	Rules    []ScheduleRule `yaml:"rules"`
}

// ScheduleRule defines when a set of profiles should stay connected.
type ScheduleRule struct {
	Name     string   `yaml:"name"`
	Profiles []string `yaml:"profiles"`
	Weekdays []string `yaml:"weekdays"` // mon,tue,wed,thu,fri,sat,sun (optional)
	Start    string   `yaml:"start"`    // HH:MM 24h (optional)
	End      string   `yaml:"end"`      // HH:MM 24h (optional)
}

// SplitDNSConfig enables transparent host-level split DNS injection.
type SplitDNSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"` // e.g. "60s"
}

// AlertsConfig defines alert thresholds and actions.
type AlertsConfig struct {
	OnVPNDrop           AlertAction `yaml:"on_vpn_drop"`
	OnLeakDetected      AlertAction `yaml:"on_leak_detected"`
	OnHighLatencyMS     int         `yaml:"on_high_latency_ms"`
	OnReconnectAttempts int         `yaml:"on_reconnect_attempts"`
}

// AlertAction specifies what to do when an alert fires.
type AlertAction struct {
	Actions []string `yaml:"actions"` // notify | log | kill_switch
}
