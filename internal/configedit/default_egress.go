package configedit

import (
	"maps"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// SetDefaultEgress applies the same edit for the daemon and offline CLI.
func SetDefaultEgress(cfg *config.Config, name string, enabled bool, save func(*config.Config) error) error {
	profile, ok := cfg.VPNs[name]
	if !ok {
		return &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}}
	}
	if profile.DefaultEgress == enabled {
		return nil
	}
	if enabled {
		if err := profile.CheckDefaultEgress(); err != nil {
			return err
		}
	}
	profile.DefaultEgress = enabled
	trial := *cfg
	trial.VPNs = maps.Clone(cfg.VPNs)
	trial.VPNs[name] = profile
	if err := save(&trial); err != nil {
		return err
	}
	cfg.VPNs[name] = profile
	return nil
}
