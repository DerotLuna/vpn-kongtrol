package config

import "strconv"

// Config is the root configuration structure for Kongtrol.
type Config struct {
	VPNs     map[string]VPNConfig   `yaml:"vpns"     validate:"required,min=1,dive"`
	Groups   map[string]GroupConfig `yaml:"groups,omitempty"`
	Policies []PolicyRule           `yaml:"policies,omitempty"`
	Security SecurityConfig         `yaml:"security,omitempty"`
	Monitor  MonitorConfig          `yaml:"monitor,omitempty"`
	Updates  UpdatesConfig          `yaml:"updates,omitempty"`
}

// UpdatesConfig is the opt-in update notice (ADR-0034). Off by default: with
// check_enabled false Kongtrol never contacts api.github.com. It only tells
// the user a newer version exists; it never downloads or installs.
type UpdatesConfig struct {
	CheckEnabled bool `yaml:"check_enabled,omitempty"`
}

// GroupConfig defines a named set of VPN profiles that can be
// connected / disconnected together with --group <name>.
type GroupConfig struct {
	Profiles []string `yaml:"profiles,omitempty" validate:"required,min=1"`
}

// VPNConfig holds the configuration for a single VPN profile.
type VPNConfig struct {
	Type       string `yaml:"type,omitempty"        validate:"required,oneof=forticlient openvpn protonvpn nordvpn surfshark ciscoanyconnect wireguard globalprotect tailscale cloudflarewarp mullvad pia expressvpn windscribe cyberghost"`
	Version    string `yaml:"version,omitempty"`
	Host       string `yaml:"host,omitempty"`
	Port       int    `yaml:"port,omitempty"`
	TunnelName string `yaml:"tunnel_name,omitempty"`
	// DefaultEgress preserves native default routing for direct WireGuard on Linux.
	DefaultEgress bool   `yaml:"default_egress,omitempty"`
	AppRouteMode  string `yaml:"app_route_mode,omitempty"`
	// FortiClient-only: when true, GUI automation accepts certificate warnings.
	// Keep false in production to avoid MITM risk.
	AllowInsecureCert bool       `yaml:"allow_insecure_cert,omitempty"`
	ConfigFile        string     `yaml:"config,omitempty"`      // .ovpn / wg .conf file path
	Server            string     `yaml:"server,omitempty"`      // server identifier (protonvpn); target country/group/server (nordvpn)
	Protocol          string     `yaml:"protocol,omitempty"`    // wireguard|openvpn (protonvpn)
	BinaryPath        string     `yaml:"binary_path,omitempty"` // override adapter binary location (optional)
	Auth              AuthConfig `yaml:"auth,omitempty"`
	// Priority orders overlapping policies (see policy.New): higher wins.
	// It never links profiles: failover only follows Failover.
	Priority   int   `yaml:"priority,omitempty"`
	KillSwitch *bool `yaml:"kill_switch,omitempty"` // optional per-profile override
	// OnTunnelDown is the default action for policies through this profile.
	OnTunnelDown string `yaml:"on_tunnel_down,omitempty"`
	// Failover lists, in order, the profiles the watchdog may connect when
	// this one drops and cannot reconnect. Empty (default): never connect
	// another profile. Profiles are never linked by name, type or priority.
	Failover []string `yaml:"failover,omitempty"`

	// Provider attach mode (forticlient, protonvpn, nordvpn, surfshark):
	// Kongtrol works with the provider's official app and attaches to its
	// tunnel by network interface. See ProviderTypes / validateProvider.

	// AutoConnect (default true) tries the official CLI or, for corporate
	// clients, GUI automation before asking the user to connect in the app.
	AutoConnect *bool `yaml:"auto_connect,omitempty"`
	// Country / City are the standard location fields, mapped to each
	// provider's syntax (country code or name; city name). Server (above)
	// names one exact server. Only for providers whose CLI can select.
	Country string `yaml:"country,omitempty"`
	City    string `yaml:"city,omitempty"`
	// Group is a NordVPN server group ("P2P", "Dedicated IP", "Double VPN",
	// "Onion Over VPN"): nordvpn only.
	Group string `yaml:"group,omitempty"`
	// Interface overrides the tunnel interface name patterns
	// (comma-separated, case-insensitive globs on name or description).
	Interface string `yaml:"interface,omitempty"`
	// WaitTimeout bounds how long `up` waits for the user to connect in the
	// app (Go duration, default 5m, minimum 10s).
	WaitTimeout string `yaml:"wait_timeout,omitempty"`
	// AppPath overrides where the provider's app is launched from.
	AppPath string `yaml:"app_path,omitempty"`
}

// ProviderTypes are the adapter types that run in attach mode over the
// provider's official app.
var ProviderTypes = map[string]bool{
	"forticlient": true, "protonvpn": true, "nordvpn": true, "surfshark": true,
	"mullvad": true, "pia": true, "expressvpn": true, "windscribe": true, "cyberghost": true,
}

// AdapterExtra returns the AdapterConfig.Extra map for this profile — the
// one place profile fields are mapped to adapter parameters.
func (v VPNConfig) AdapterExtra() map[string]string {
	extra := map[string]string{
		"protocol":            v.Protocol,
		"allow_insecure_cert": strconv.FormatBool(v.AllowInsecureCert),
		"binary_path":         v.BinaryPath,
		"server":              v.Server,
		"country":             v.Country,
		"city":                v.City,
		"group":               v.Group,
		"interface":           v.Interface,
		"wait_timeout":        v.WaitTimeout,
		"app_path":            v.AppPath,
	}
	if v.AutoConnect != nil {
		extra["auto_connect"] = strconv.FormatBool(*v.AutoConnect)
	}
	return extra
}

// AuthConfig defines authentication credentials for a VPN profile.
// Passwords are never stored here — they live in the OS keychain.
type AuthConfig struct {
	// certificate | credentials | certificate+credentials
	Method           string `yaml:"method,omitempty" validate:"required"`
	Cert             string `yaml:"cert,omitempty"`              // path to client certificate
	Key              string `yaml:"key,omitempty"`               // path to private key
	Username         string `yaml:"username,omitempty"`          // plaintext username (low-sensitivity)
	UsernameKeychain string `yaml:"username_keychain,omitempty"` // keychain key name for username
	PasswordKeychain string `yaml:"password_keychain,omitempty"` // keychain key name for password
}

// PolicyRule maps traffic to a specific VPN profile.
type PolicyRule struct {
	Name  string    `yaml:"name,omitempty" validate:"required"`
	Match MatchSpec `yaml:"match,omitempty"`
	Via   string    `yaml:"via,omitempty"  validate:"required"`
	// Disabled saves a rule without routing, blocking or waking a tunnel.
	Disabled bool `yaml:"disabled,omitempty"`
	// OnTunnelDown is what happens to this policy's traffic while its
	// tunnel is not connected (the user decides which tunnels are up):
	//   allow   normal internet, and Kongtrol notifies;
	//   block   the traffic must not leave through the normal internet;
	//   connect block and try connecting when TCP activity is observed.
	// An empty value inherits the profile action.
	OnTunnelDown string `yaml:"on_tunnel_down,omitempty"`
}

// ActivePolicies returns the rules that participate in routing, in file order.
// Paused rules must stay out of the positional priority calculation.
func (c *Config) ActivePolicies() []PolicyRule {
	active := make([]PolicyRule, 0, len(c.Policies))
	for _, rule := range c.Policies {
		if !rule.Disabled {
			active = append(active, rule)
		}
	}
	return active
}

// On-tunnel-down actions (PolicyRule.OnTunnelDown).
const (
	OnTunnelDownAllow   = "allow"
	OnTunnelDownBlock   = "block"
	OnTunnelDownConnect = "connect"
)

// BlocksWhenDown reports whether an action requires firewall protection.
func BlocksWhenDown(action string) bool {
	return action == OnTunnelDownBlock || action == OnTunnelDownConnect
}

// TunnelDownAction returns the profile default, or allow when omitted.
func (v VPNConfig) TunnelDownAction() string {
	if v.OnTunnelDown == "" {
		return OnTunnelDownAllow
	}
	return v.OnTunnelDown
}

// TunnelDownAction returns the explicit policy action or the profile default.
func (p PolicyRule) TunnelDownAction(profile VPNConfig) string {
	if p.OnTunnelDown == "" {
		return profile.TunnelDownAction()
	}
	return p.OnTunnelDown
}

// MatchSpec defines which traffic a policy rule matches.
type MatchSpec struct {
	IPRanges []string `yaml:"ip_ranges,omitempty"` // CIDR notation
	Domains  []string `yaml:"domains,omitempty"`   // glob patterns (*.example.com)
	Apps     []string `yaml:"apps,omitempty"`      // executable names or glob patterns (experimental)
}

// SecurityConfig holds all security-related settings.
type SecurityConfig struct {
	KillSwitch     KillSwitchConfig     `yaml:"kill_switch,omitempty"`
	DNSGuard       DNSGuardConfig       `yaml:"dns_guard,omitempty"`
	IntegrityCheck IntegrityCheckConfig `yaml:"integrity_check,omitempty"`
	LeakDetection  LeakDetectionConfig  `yaml:"leak_detection,omitempty"`
	PublicIPCheck  PublicIPCheckConfig  `yaml:"public_ip_check,omitempty"`
	AuditLog       AuditLogConfig       `yaml:"audit_log,omitempty"`
}

// KillSwitchConfig configures the kill switch behavior.
//
// Split-tunnel semantics: the kill switch never blocks a tunnel the user
// turned off (`down`). When a protected tunnel drops unexpectedly, only
// that tunnel's destinations (routes captured while it was up + its
// policies' IPs/CIDRs) are blocked until it reconnects or the user runs
// `down`; the rest of the internet keeps working. A full-tunnel profile
// (default route through the tunnel) blocks everything except the VPN
// endpoints (and the LAN when allowed) while dropped.
//
// Mode stays for backward compatibility: "loose" now means the same as
// allow_lan: true (LAN reachable while a full tunnel is dropped);
// "strict" or empty leaves it to allow_lan.
type KillSwitchConfig struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Mode     string `yaml:"mode,omitempty" validate:"omitempty,oneof=strict loose"`
	AllowLAN bool   `yaml:"allow_lan,omitempty"`
}

// LANAllowed reports whether the LAN stays reachable while a full-tunnel
// profile is blocked (allow_lan, or the legacy mode: loose).
func (k KillSwitchConfig) LANAllowed() bool { return k.AllowLAN || k.Mode == "loose" }

// DNSGuardConfig configures DNS leak prevention.
type DNSGuardConfig struct {
	Enabled     bool   `yaml:"enabled,omitempty"`
	FallbackDNS string `yaml:"fallback_dns,omitempty"`
}

// IntegrityCheckConfig configures tunnel integrity verification.
type IntegrityCheckConfig struct {
	Enabled     bool              `yaml:"enabled,omitempty"`
	Interval    string            `yaml:"interval,omitempty"`     // e.g. "120s"
	ExpectedIPs map[string]string `yaml:"expected_ips,omitempty"` // vpn_name → expected public IP
}

// LeakDetectionConfig is the legacy leak detection block. Leak detection
// compares the public IP (see PublicIPCheckConfig) with
// integrity_check.expected_ips, so it needs the public IP lookup:
// enabled: true here still counts as opting in to that lookup
// (Config.PublicIPCheckEnabled). A detected leak notifies (alert, audit
// entry, dashboards); it never drives the kill switch (ADR-0042). The old
// interval and action keys are gone: a config that still has them loads
// and they are dropped on the next save.
type LeakDetectionConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// PublicIPCheckConfig configures the opt-in public IP lookup: an HTTPS GET
// to an external service that answers with the caller's public IP in plain
// text. Off by default (Local only rule): that service sees the user's
// public IP. It runs only on tunnel connect/disconnect (rate-limited) and
// when the user asks (`kongtrol ip`, dashboard "Revisar", tray).
type PublicIPCheckConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
	// Endpoints are tried in order until one answers with a valid IP.
	// Empty = DefaultPublicIPEndpoint. Each must be an https:// URL.
	Endpoints []string `yaml:"endpoints,omitempty"`
}

// AuditLogConfig configures the tamper-evident audit log.
type AuditLogConfig struct {
	Path string `yaml:"path,omitempty"`
	// MaxSizeMB is the largest size of one audit file (MB): past it the
	// same day continues in "<base>-YYYY-MM-DD.1<ext>", ".2"… (default 100).
	MaxSizeMB int `yaml:"max_size_mb,omitempty"`
	// RetentionDays keeps the files of the last N days, today included
	// (0 = DefaultAuditRetentionDays; at most 365). Older days are deleted
	// at start, when a file fills up and every midnight (ADR-0064).
	RetentionDays int `yaml:"retention_days,omitempty"`
	// MaxTotalMB caps the space of all audit files together (MB): the
	// oldest are deleted first (0 = DefaultAuditMaxTotalMB, never below
	// MaxSizeMB; at most 10240).
	MaxTotalMB int  `yaml:"max_total_mb,omitempty"`
	Sign       bool `yaml:"sign,omitempty"` // HMAC-SHA256 per entry
}

// Audit log limits (ADR-0064). internal/security keeps the same defaults
// for a logger opened without a config (a test checks they match).
const (
	DefaultAuditRetentionDays = 30
	MaxAuditRetentionDays     = 365
	DefaultAuditMaxSizeMB     = 100
	MaxAuditMaxSizeMB         = 1024
	DefaultAuditMaxTotalMB    = 500
	MaxAuditMaxTotalMB        = 10240
)

// EffectiveRetentionDays is the number of days really kept: 0 (unset) is
// DefaultAuditRetentionDays.
func (a AuditLogConfig) EffectiveRetentionDays() int {
	if a.RetentionDays <= 0 {
		return DefaultAuditRetentionDays
	}
	return min(a.RetentionDays, MaxAuditRetentionDays)
}

// EffectiveMaxSizeMB is the largest size of one file: 0 (unset) is
// DefaultAuditMaxSizeMB.
func (a AuditLogConfig) EffectiveMaxSizeMB() int {
	if a.MaxSizeMB <= 0 {
		return DefaultAuditMaxSizeMB
	}
	return min(a.MaxSizeMB, MaxAuditMaxSizeMB)
}

// EffectiveMaxTotalMB is the space all files may use together: 0 (unset)
// is DefaultAuditMaxTotalMB, raised to one file's size when that is larger
// (so a config written before max_total_mb existed keeps working).
func (a AuditLogConfig) EffectiveMaxTotalMB() int {
	if a.MaxTotalMB <= 0 {
		return max(DefaultAuditMaxTotalMB, a.EffectiveMaxSizeMB())
	}
	return min(a.MaxTotalMB, MaxAuditMaxTotalMB)
}

// MonitorConfig holds monitoring and dashboard settings.
type MonitorConfig struct {
	Enabled     bool              `yaml:"enabled,omitempty"`
	Dashboard   DashboardConfig   `yaml:"dashboard,omitempty"`
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty"`
	History     HistoryConfig     `yaml:"history,omitempty"`
	Scheduler   SchedulerConfig   `yaml:"scheduler,omitempty"`
	SplitDNS    SplitDNSConfig    `yaml:"split_dns,omitempty"`
	Alerts      AlertsConfig      `yaml:"alerts,omitempty"`
	// RecentConnections: the in-memory list of recent destinations
	// (GET /api/v1/connections/recent, `kongtrol traffic recent`, ADR-0054).
	RecentConnections RecentConnectionsConfig `yaml:"recent_connections,omitempty"`
}

// RecentConnectionsConfig turns the recent destinations list on or off.
// Enabled nil means on (the default): the list only lives in the daemon's
// memory and is never written to disk, logged or sent anywhere.
type RecentConnectionsConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}

// DashboardConfig configures the embedded web dashboard.
type DashboardConfig struct {
	Port int    `yaml:"port,omitempty"` // default 9741
	Bind string `yaml:"bind,omitempty"` // default 127.0.0.1
}

// HealthCheckConfig configures per-tunnel health checks.
type HealthCheckConfig struct {
	Interval string `yaml:"interval,omitempty"` // e.g. "30s"
	Timeout  string `yaml:"timeout,omitempty"`  // e.g. "10s"
}

// HistoryConfig stores rolling profile stability metrics.
type HistoryConfig struct {
	Path          string `yaml:"path,omitempty"`           // e.g. ~/.kongtrol/history.json
	FlushInterval string `yaml:"flush_interval,omitempty"` // e.g. "30s"
}

// SchedulerConfig activates profile groups by schedule windows.
type SchedulerConfig struct {
	Enabled  bool           `yaml:"enabled,omitempty"`
	Interval string         `yaml:"interval,omitempty"` // e.g. "1m"
	Rules    []ScheduleRule `yaml:"rules,omitempty"`
}

// ScheduleRule defines when a set of profiles should stay connected.
type ScheduleRule struct {
	Name     string   `yaml:"name,omitempty"`
	Profiles []string `yaml:"profiles,omitempty"`
	Weekdays []string `yaml:"weekdays,omitempty"` // mon,tue,wed,thu,fri,sat,sun (optional)
	Start    string   `yaml:"start,omitempty"`    // HH:MM 24h (optional)
	End      string   `yaml:"end,omitempty"`      // HH:MM 24h (optional)
	// Enabled pauses a rule without deleting it: false = the scheduler
	// ignores it. Unset (default) = on.
	Enabled *bool `yaml:"enabled,omitempty"`
}

// IsEnabled reports whether the scheduler applies this rule (default on).
func (r ScheduleRule) IsEnabled() bool { return r.Enabled == nil || *r.Enabled }

// SplitDNSConfig enables transparent host-level split DNS injection.
type SplitDNSConfig struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Interval string `yaml:"interval,omitempty"` // e.g. "60s"
}

// AlertsConfig defines alert thresholds and actions.
type AlertsConfig struct {
	OnVPNDrop           AlertAction `yaml:"on_vpn_drop,omitempty"`
	OnLeakDetected      AlertAction `yaml:"on_leak_detected,omitempty"`
	OnHighLatencyMS     int         `yaml:"on_high_latency_ms,omitempty"`
	OnReconnectAttempts int         `yaml:"on_reconnect_attempts,omitempty"`
}

// AlertAction specifies what to do when an alert fires.
type AlertAction struct {
	Actions []string `yaml:"actions,omitempty"` // notify | log | kill_switch
}

// WithDefaults fills profile defaults that may be left out of the YAML. App
// providers whose login lives in the official app (protonvpn, nordvpn,
// surfshark) need no auth block: auth.method defaults to "credentials".
func (v VPNConfig) WithDefaults() VPNConfig {
	if v.Auth.Method == "" && ProviderTypes[v.Type] && v.Type != "forticlient" {
		v.Auth.Method = "credentials"
	}
	return v
}
