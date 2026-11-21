package configedit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ErrKeyVPNExists is the key of "a profile with that name exists"; the API
// answers it with 409.
const ErrKeyVPNExists = "api.err.vpn_exists"

// VPNProfile is the create/update body of a VPN profile. Every field is a
// pointer so a field absent from the JSON (nil) is distinguishable from one
// sent empty: on update only the fields present are applied on top of the
// existing VPNConfig, so YAML-only fields the dashboard form does not expose
// (binary_path, allow_insecure_cert, version, tunnel_name, ...) survive.
type VPNProfile struct {
	Name         *string `json:"name"`
	Type         *string `json:"type"`
	Host         *string `json:"host"`
	Port         *int    `json:"port"`
	Server       *string `json:"server"`
	Protocol     *string `json:"protocol"`
	ConfigFile   *string `json:"config"`
	Priority     *int    `json:"priority"`
	OnTunnelDown *string `json:"on_tunnel_down"`
	// Failover: the backup profiles, in order; [] clears it.
	Failover   *[]string `json:"failover"`
	AuthMethod *string   `json:"auth_method"`
	Cert       *string   `json:"cert"`
	Key        *string   `json:"key"`
	Username   *string   `json:"username"`
	// Provider attach mode. AutoConnect true is the default and is stored
	// as "unset" so the YAML stays minimal.
	AutoConnect *bool   `json:"auto_connect"`
	Country     *string `json:"country"`
	City        *string `json:"city"`
	Group       *string `json:"group"`
	Interface   *string `json:"interface"`
	WaitTimeout *string `json:"wait_timeout"`
	AppPath     *string `json:"app_path"`
	// Password is plaintext in the request only; it is written to the OS
	// keychain and never persisted to the YAML config.
	Password string `json:"password,omitempty"`
}

// Str returns the trimmed value of p, or "" when absent.
func Str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func setTrimmed(dst, src *string) {
	if src != nil {
		*dst = strings.TrimSpace(*src)
	}
}

// ApplyTo overlays the fields present in p onto vc.
func (p VPNProfile) ApplyTo(vc *config.VPNConfig) {
	if p.OnTunnelDown != nil {
		vc.OnTunnelDown = strings.ToLower(Str(p.OnTunnelDown))
	}
	setTrimmed(&vc.Type, p.Type)
	setTrimmed(&vc.Host, p.Host)
	setTrimmed(&vc.Server, p.Server)
	setTrimmed(&vc.Protocol, p.Protocol)
	setTrimmed(&vc.ConfigFile, p.ConfigFile)
	setTrimmed(&vc.Auth.Method, p.AuthMethod)
	setTrimmed(&vc.Auth.Cert, p.Cert)
	setTrimmed(&vc.Auth.Key, p.Key)
	setTrimmed(&vc.Auth.Username, p.Username)
	setTrimmed(&vc.Country, p.Country)
	setTrimmed(&vc.City, p.City)
	setTrimmed(&vc.Group, p.Group)
	setTrimmed(&vc.Interface, p.Interface)
	setTrimmed(&vc.WaitTimeout, p.WaitTimeout)
	setTrimmed(&vc.AppPath, p.AppPath)
	if p.AutoConnect != nil {
		if *p.AutoConnect {
			vc.AutoConnect = nil
		} else {
			off := false
			vc.AutoConnect = &off
		}
	}
	if p.Port != nil {
		vc.Port = *p.Port
	}
	if p.Priority != nil {
		vc.Priority = *p.Priority
	}
	if p.Failover != nil {
		vc.Failover = nil
		for _, f := range *p.Failover {
			if f = strings.TrimSpace(f); f != "" {
				vc.Failover = append(vc.Failover, f)
			}
		}
	}
}

// AddVPNProfile creates the profile named in p (it must be new): see
// SaveVPNProfile. It returns the profile name.
func AddVPNProfile(cfg *config.Config, p VPNProfile, save func(*config.Config) error) (string, error) {
	name := Str(p.Name)
	if name == "" {
		return "", &config.ValidationError{Key: "api.err.name_required"}
	}
	if _, exists := cfg.VPNs[name]; exists {
		return name, &config.ValidationError{Key: ErrKeyVPNExists, Args: []any{name}}
	}
	return name, SaveVPNProfile(cfg, name, config.VPNConfig{}, p, save)
}

// SaveVPNProfile overlays p onto base (zero value on create, the stored
// profile on update), validates the result against a trial copy of cfg
// before writing anything, then stores any provided credentials in the OS
// keychain and persists cfg with save (the daemon's save + hot swap, or
// Save for the CLI). A failed save puts the previous credentials back.
func SaveVPNProfile(cfg *config.Config, name string, base config.VPNConfig, p VPNProfile, save func(*config.Config) error) error {
	for field, value := range map[string]*string{
		"name": p.Name, "type": p.Type, "host": p.Host, "server": p.Server,
		"protocol": p.Protocol, "config": p.ConfigFile, "auth_method": p.AuthMethod,
		"cert": p.Cert, "key": p.Key, "username": p.Username,
		"country": p.Country, "city": p.City, "group": p.Group, "interface": p.Interface,
		"wait_timeout": p.WaitTimeout, "app_path": p.AppPath,
	} {
		if value != nil && containsEnvReference(*value) {
			return fmt.Errorf("%s must not contain environment-variable references", field)
		}
	}

	vc := base
	p.ApplyTo(&vc)
	vc = vc.WithDefaults()
	username := Str(p.Username)
	if username != "" {
		vc.Auth.UsernameKeychain = name + ".username"
	}
	if p.Password != "" {
		vc.Auth.PasswordKeychain = name + ".password"
	}

	trial := *cfg
	trialVPNs := make(map[string]config.VPNConfig, len(cfg.VPNs)+1)
	for k, v := range cfg.VPNs {
		trialVPNs[k] = v
	}
	trialVPNs[name] = vc
	trial.VPNs = trialVPNs
	if err := config.Validate(&trial); err != nil {
		return err
	}
	// A config file only root can read would fail at every connect:
	// refuse it now with the way out (copy it with permission, ADR-0058).
	if p.ConfigFile != nil {
		if err := config.CheckFileReadable(vc.ConfigFile); err != nil {
			return err
		}
	}

	if cfg.VPNs == nil {
		cfg.VPNs = map[string]config.VPNConfig{}
	}
	cfg.VPNs[name] = vc
	backups := make([]credentialBackup, 0, 2)
	if username != "" {
		backup := backupCredential(name, "username")
		if err := config.SetCredential(name, "username", username); err != nil {
			return fmt.Errorf("store username credential: %w", err)
		}
		backups = append(backups, backup)
	}
	if p.Password != "" {
		backup := backupCredential(name, "password")
		if err := config.SetCredential(name, "password", p.Password); err != nil {
			return errors.Join(
				fmt.Errorf("store password credential: %w", err),
				restoreCredentials(name, backups),
			)
		}
		backups = append(backups, backup)
	}
	if err := save(cfg); err != nil {
		return errors.Join(err, restoreCredentials(name, backups))
	}
	return nil
}

type credentialBackup struct {
	key     string
	value   string
	existed bool
}

func backupCredential(profile, key string) credentialBackup {
	value, err := config.GetCredential(profile, key)
	return credentialBackup{key: key, value: value, existed: err == nil}
}

func restoreCredentials(profile string, backups []credentialBackup) error {
	var errs []error
	for i := len(backups) - 1; i >= 0; i-- {
		backup := backups[i]
		var err error
		if backup.existed {
			err = config.SetCredential(profile, backup.key, backup.value)
		} else {
			err = config.DeleteCredential(profile, backup.key)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("restore %s credential: %w", backup.key, err))
		}
	}
	return errors.Join(errs...)
}

func containsEnvReference(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] != '$' || i+1 >= len(value) {
			continue
		}
		next := value[i+1]
		if next == '{' || next == '_' || next >= 'A' && next <= 'Z' || next >= 'a' && next <= 'z' {
			return true
		}
	}
	return false
}

// DefaultAuthMethod is the auth method a new profile of type typ gets when
// none is chosen, as in the dashboard's catalog (web/ui
// components/profiles/catalog.ts defaultAuth): certificate for the
// config-file types, credentials for the rest.
func DefaultAuthMethod(typ string) string {
	switch typ {
	case "wireguard", "openvpn":
		return "certificate"
	}
	return "credentials"
}

// NeedsPassword reports whether a profile of type typ with auth method
// method asks for a password stored in the keychain: corporate clients and
// OpenVPN whose method includes credentials (dashboard catalog auth
// "main"). Provider apps keep their own login; Tailscale/WARP use none.
func NeedsPassword(typ, method string) bool {
	switch typ {
	case "forticlient", "openvpn", "ciscoanyconnect", "globalprotect":
		return strings.Contains(method, "credentials")
	}
	return false
}
