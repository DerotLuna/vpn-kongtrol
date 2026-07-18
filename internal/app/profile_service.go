package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

type Watchdog interface {
	MarkIntended(name string)
	MarkActive(name string)
}

type PolicyResolver interface {
	RegisterProfile(name, ifaceName, configPath string) error
	UnregisterProfile(name string)
}

type DNSManager interface {
	OnConnect(profile string, iface string, dnsServers []net.IP)
	OnDisconnect(profile string)
}

type ProfileService struct {
	cfg            *config.Config
	adapters       map[string]vpn.VPNAdapter
	watchdog       Watchdog
	policyResolver PolicyResolver
	dnsMgr         DNSManager
	applyKill      func() error
	format         func(key string, args ...any) string
	translate      func(key string) string
	emitAlert      func(level, profile, message string)
	logAudit       func(level, action, profile, message string)
	emitWarn       func(message string)
	emitStderr     func(message string)
}

type ProfileServiceDeps struct {
	Cfg            *config.Config
	Adapters       map[string]vpn.VPNAdapter
	Watchdog       Watchdog
	PolicyResolver PolicyResolver
	DNSManager     DNSManager
	ApplyKill      func() error
	Format         func(key string, args ...any) string
	Translate      func(key string) string
	EmitAlert      func(level, profile, message string)
	LogAudit       func(level, action, profile, message string)
	EmitWarn       func(message string)
	EmitStderr     func(message string)
}

func NewProfileService(deps ProfileServiceDeps) *ProfileService {
	return &ProfileService{
		cfg:            deps.Cfg,
		adapters:       deps.Adapters,
		watchdog:       deps.Watchdog,
		policyResolver: deps.PolicyResolver,
		dnsMgr:         deps.DNSManager,
		applyKill:      deps.ApplyKill,
		format:         deps.Format,
		translate:      deps.Translate,
		emitAlert:      deps.EmitAlert,
		logAudit:       deps.LogAudit,
		emitWarn:       deps.EmitWarn,
		emitStderr:     deps.EmitStderr,
	}
}

func (s *ProfileService) Connect(ctx context.Context, name string) error {
	adapter, ok := s.adapters[name]
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.unknown_profile", name))
	}

	vpnCfg, ok := s.cfg.VPNs[name]
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.no_config_for_profile", name))
	}

	aCfg := vpn.AdapterConfig{
		Host:       vpnCfg.Host,
		Port:       vpnCfg.Port,
		TunnelName: vpnCfg.TunnelName,
		CertPath:   vpnCfg.Auth.Cert,
		KeyPath:    vpnCfg.Auth.Key,
		ConfigPath: vpnCfg.ConfigFile,
		Username:   vpnCfg.Auth.Username,
		Extra: map[string]string{
			"protocol":            vpnCfg.Protocol,
			"allow_insecure_cert": strconv.FormatBool(vpnCfg.AllowInsecureCert),
		},
	}

	if vpnCfg.Auth.UsernameKeychain != "" {
		u, err := config.GetCredential(name, "username")
		if err == nil {
			aCfg.Username = u
		}
	}
	if vpnCfg.Auth.PasswordKeychain != "" {
		p, err := config.GetCredential(name, "password")
		if err == nil {
			aCfg.Password = p
		}
	}

	if err := s.runProfilePreflight(name, vpnCfg, aCfg); err != nil {
		aCfg.Password = ""
		s.audit("ERROR", "vpn.preflight_failed", name, s.f("cli.audit.preflight_failed", name, err))
		return err
	}

	if vpnCfg.Type == "wireguard" && aCfg.ConfigPath != "" {
		if cidrs := s.policyAllowedIPs(name); len(cidrs) > 0 {
			if aCfg.TunnelName == "" {
				base := filepath.Base(aCfg.ConfigPath)
				aCfg.TunnelName = strings.TrimSuffix(base, filepath.Ext(base))
			}
			patched, err := patchWireGuardAllowedIPs(aCfg.ConfigPath, aCfg.TunnelName, cidrs)
			if err != nil {
				s.stderr(s.f("cli.policy.warn.patch_allowed_ips", name, err))
			} else {
				aCfg.ConfigPath = patched
				defer os.RemoveAll(filepath.Dir(patched))
			}
		}
	}

	if adapter.Status().Normalize() != vpn.StatusConnected {
		err := adapter.Connect(ctx, aCfg)
		aCfg.Password = ""
		if err != nil {
			s.maybeEmitReauthAssistance(name, err)
			s.audit("ERROR", "vpn.connect_failed", name, s.f("cli.audit.connect_failed", name, err))
			return err
		}
	}

	if s.watchdog != nil {
		s.watchdog.MarkActive(name)
	}

	if s.dnsMgr != nil && s.cfg.Security.DNSGuard.Enabled && vpnCfg.Type != "wireguard" {
		if info, err := adapter.TunnelInfo(); err == nil && info != nil {
			dnsServers := info.DNS
			if len(dnsServers) == 0 && s.cfg.Security.DNSGuard.FallbackDNS != "" {
				if ip := net.ParseIP(s.cfg.Security.DNSGuard.FallbackDNS); ip != nil {
					dnsServers = []net.IP{ip}
				}
			}
			if len(dnsServers) > 0 {
				s.dnsMgr.OnConnect(name, info.InterfaceName, dnsServers)
			}
		}
	}

	if s.policyResolver != nil && vpnCfg.Type == "wireguard" {
		ifaceName := interfaceFromWGConfig(vpnCfg)
		if err := s.policyResolver.RegisterProfile(name, ifaceName, vpnCfg.ConfigFile); err != nil {
			s.stderr(s.f("cli.policyresolver.warn", name, err))
		}
	}
	if s.applyKill != nil {
		_ = s.applyKill()
	}

	s.audit("INFO", "vpn.connect", name, s.f("cli.audit.connect", name))
	return nil
}

func (s *ProfileService) Disconnect(ctx context.Context, name string) error {
	adapter, ok := s.adapters[name]
	if !ok {
		return fmt.Errorf("%s", s.f("cli.error.unknown_profile", name))
	}
	if s.watchdog != nil {
		s.watchdog.MarkIntended(name)
	}
	if s.policyResolver != nil {
		s.policyResolver.UnregisterProfile(name)
	}
	if err := adapter.Disconnect(ctx); err != nil {
		return err
	}
	if s.dnsMgr != nil {
		s.dnsMgr.OnDisconnect(name)
	}
	if s.applyKill != nil {
		_ = s.applyKill()
	}
	s.audit("INFO", "vpn.disconnect", name, s.f("cli.audit.disconnect", name))
	return nil
}

func (s *ProfileService) HandleReconnectError(profile string, err error) {
	s.maybeEmitReauthAssistance(profile, err)
}

func (s *ProfileService) PolicyAllowedIPs(profileName string) []string {
	return s.policyAllowedIPs(profileName)
}

func (s *ProfileService) runProfilePreflight(name string, vpnCfg config.VPNConfig, aCfg vpn.AdapterConfig) error {
	if vpnCfg.ConfigFile != "" {
		if _, err := os.Stat(vpnCfg.ConfigFile); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_config", name, err))
		}
	}
	if vpnCfg.Auth.Cert != "" {
		if _, err := os.Stat(vpnCfg.Auth.Cert); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_cert", name, err))
		}
	}
	if vpnCfg.Auth.Key != "" {
		if _, err := os.Stat(vpnCfg.Auth.Key); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_key", name, err))
		}
	}
	if vpnCfg.Auth.PasswordKeychain != "" && strings.TrimSpace(aCfg.Password) == "" {
		return fmt.Errorf("%s", s.f("cli.preflight.missing_password", name))
	}
	if vpnCfg.BinaryPath != "" {
		if _, err := os.Stat(vpnCfg.BinaryPath); err != nil {
			return fmt.Errorf("%s", s.f("cli.preflight.missing_binary", name, vpnCfg.BinaryPath))
		}
	} else {
		if bin := adapterBinaryByType(vpnCfg.Type); bin != "" {
			if _, err := exec.LookPath(bin); err != nil {
				s.warn(s.f("cli.preflight.binary_warning", name, bin))
			}
		}
	}
	if aCfg.Host != "" {
		port := aCfg.Port
		if port == 0 {
			port = 443
		}
		dialer := net.Dialer{Timeout: 2 * time.Second}
		conn, err := dialer.Dial("tcp", net.JoinHostPort(aCfg.Host, strconv.Itoa(port)))
		if err != nil {
			s.warn(s.f("cli.preflight.port_warning", name, aCfg.Host, port, err))
		} else {
			_ = conn.Close()
		}
	}
	return nil
}

func adapterBinaryByType(adapterType string) string {
	switch adapterType {
	case "openvpn":
		return "openvpn"
	case "protonvpn":
		return "protonvpn-cli"
	case "tailscale":
		return "tailscale"
	case "cloudflarewarp":
		return "warp-cli"
	case "wireguard":
		return "wg"
	case "forticlient":
		return "forticlientsslvpn"
	case "ciscoanyconnect":
		return "vpn"
	default:
		return ""
	}
}

func (s *ProfileService) maybeEmitReauthAssistance(profile string, err error) {
	if err == nil || !isReauthError(err) {
		return
	}
	vpnCfg, ok := s.cfg.VPNs[profile]
	if !ok {
		return
	}
	hint := s.reauthHint(vpnCfg)
	msg := s.f("cli.alert.reauth_required", profile, err)
	if s.emitAlert != nil {
		s.emitAlert("WARN", profile, msg)
		if hint != "" {
			s.emitAlert("INFO", profile, s.f("cli.alert.reauth_hint", profile, hint))
		}
	}
	s.audit("WARN", "vpn.reauth_required", profile, msg)
}

func isReauthError(err error) bool {
	msg := strings.ToLower(err.Error())
	needles := []string{
		"unauthorized", "authorization", "authentication", "auth failed",
		"invalid credentials", "credential", "password", "token expired",
		"session expired", "login failed", "mfa",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func (s *ProfileService) reauthHint(vpnCfg config.VPNConfig) string {
	switch vpnCfg.Type {
	case "forticlient", "globalprotect", "ciscoanyconnect":
		return s.t("cli.reauth.hint.gui")
	case "openvpn", "wireguard":
		return s.t("cli.reauth.hint.credentials")
	case "protonvpn", "cloudflarewarp", "tailscale":
		return s.t("cli.reauth.hint.session")
	default:
		return s.t("cli.reauth.hint.generic")
	}
}

func (s *ProfileService) policyAllowedIPs(profileName string) []string {
	vpnCfg, ok := s.cfg.VPNs[profileName]
	if !ok {
		return nil
	}

	seen := make(map[string]struct{})
	var cidrs []string
	add := func(cidr string) {
		if cidr == "" {
			return
		}
		if _, ok := seen[cidr]; !ok {
			seen[cidr] = struct{}{}
			cidrs = append(cidrs, cidr)
		}
	}

	var endpointIP net.IP
	if vpnCfg.ConfigFile != "" {
		endpointIP, _ = wireguard.ParseEndpoint(vpnCfg.ConfigFile)
		if addr, err := wireguard.ParseConfigAddress(vpnCfg.ConfigFile); err == nil && addr != nil {
			if v4 := addr.To4(); v4 != nil {
				add(fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2]))
			}
		}
		for _, dns := range wireguard.ParseConfigDNS(vpnCfg.ConfigFile) {
			if dns.To4() != nil {
				add(dns.String() + "/32")
			}
		}
	}

	hasDomains := false
	for _, pol := range s.cfg.Policies {
		if pol.Via != profileName {
			continue
		}
		for _, cidr := range pol.Match.IPRanges {
			add(cidr)
		}
		for _, domain := range pol.Match.Domains {
			hasDomains = true
			var lookups []string
			if strings.HasPrefix(domain, "*.") {
				base := strings.TrimPrefix(domain, "*.")
				lookups = append(lookups, base)
				for _, prefix := range []string{"www.", "api.", "cdn.", "app.", "docs.", "console."} {
					lookups = append(lookups, prefix+base)
				}
			} else {
				lookups = []string{domain}
			}

			for _, lookup := range lookups {
				ips, err := net.LookupHost(lookup)
				if err != nil {
					continue
				}
				for _, ip := range ips {
					parsed := net.ParseIP(ip)
					if parsed == nil {
						continue
					}
					if endpointIP != nil && parsed.Equal(endpointIP) {
						continue
					}
					if parsed.To4() != nil {
						add(ip + "/32")
					}
				}
			}
		}
	}

	if !hasDomains && len(cidrs) == 0 {
		return nil
	}
	return cidrs
}

func interfaceFromWGConfig(vpnCfg config.VPNConfig) string {
	if vpnCfg.TunnelName != "" {
		return vpnCfg.TunnelName
	}
	base := filepath.Base(vpnCfg.ConfigFile)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func patchWireGuardAllowedIPs(configPath, tunnelName string, cidrs []string) (string, error) {
	original, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}

	allowedLine := "AllowedIPs = " + strings.Join(cidrs, ", ")
	var out strings.Builder
	inPeer := false
	replaced := false
	for _, line := range strings.Split(string(original), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPeer = strings.EqualFold(trimmed, "[Peer]")
			replaced = false
		}
		if inPeer && strings.HasPrefix(strings.ToLower(trimmed), "allowedips") {
			if !replaced {
				out.WriteString(allowedLine + "\n")
				replaced = true
			}
			continue
		}
		out.WriteString(line + "\n")
	}

	tmpDir, err := os.MkdirTemp("", "kongtrol-wg-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	tmpPath := filepath.Join(tmpDir, tunnelName+".conf")
	if err := os.WriteFile(tmpPath, []byte(out.String()), 0o600); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("write temp config: %w", err)
	}
	return tmpPath, nil
}

func (s *ProfileService) f(key string, args ...any) string {
	if s.format == nil {
		return key
	}
	return s.format(key, args...)
}

func (s *ProfileService) t(key string) string {
	if s.translate == nil {
		return key
	}
	return s.translate(key)
}

func (s *ProfileService) audit(level, action, profile, message string) {
	if s.logAudit != nil {
		s.logAudit(level, action, profile, message)
	}
}

func (s *ProfileService) warn(message string) {
	if s.emitWarn != nil {
		s.emitWarn(message)
	}
}

func (s *ProfileService) stderr(message string) {
	if s.emitStderr != nil {
		s.emitStderr(message)
	}
}
