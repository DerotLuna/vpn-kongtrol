package config

import "github.com/vpn-kongtrol/kongtrol/internal/privops"

// DefaultEgressStatus reports the profile's selection and file compatibility.
type DefaultEgressStatus struct {
	Enabled bool `json:"enabled"`
	privops.DefaultEgressCheck
}

func (v VPNConfig) DefaultEgressStatus() DefaultEgressStatus {
	return DefaultEgressStatus{
		Enabled:            v.DefaultEgress,
		DefaultEgressCheck: privops.CheckDefaultEgress(v.Type, v.ConfigFile, v.TunnelName),
	}
}

// CheckDefaultEgress validates the current profile file without touching the network.
func (v VPNConfig) CheckDefaultEgress() error {
	if result := v.DefaultEgressStatus(); !result.Compatible {
		return newValidationError("config.err.default_egress_" + result.Reason)
	}
	return nil
}

func validateDefaultEgress(profiles map[string]VPNConfig, names []string) error {
	selected := ""
	for _, name := range names {
		profile := profiles[name]
		if !profile.DefaultEgress {
			continue
		}
		if profile.Type != "wireguard" {
			return newValidationError("config.err.default_egress_type", name)
		}
		if selected != "" {
			return newValidationError("config.err.default_egress_multiple", selected, name)
		}
		selected = name
	}
	return nil
}
