package api

import (
	"reflect"
	"slices"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// ProfileSyncResult is what the daemon did with the profiles of a saved
// config (ADR-0053). Names not listed were applied live.
type ProfileSyncResult struct {
	// Reconnect: edited while the tunnel is up; the new settings apply on
	// the next connect (after the user reconnects).
	Reconnect []string
	// Kept: removed from the config while the tunnel is up (hand edit);
	// dropped once disconnected.
	Kept []string
	// Failed: no adapter could be built (unknown type); a restart would
	// fail the same way.
	Failed []string
}

// SetProfileSync lets VPN profile changes apply without a restart: fn
// brings the daemon's adapter set in line with cfg. Without it (tests,
// older wiring) every profile change answers restart_required.
func (s *Server) SetProfileSync(fn func(*config.Config) ProfileSyncResult) {
	s.profileSync = fn
}

// profileApply is how one config save reached the running daemon.
type profileApply struct {
	live bool // false: no live apply here, a restart is needed
	res  ProfileSyncResult
}

func (s *Server) syncProfiles(cfg *config.Config) profileApply {
	// Setup mode keeps its restart: the daemon leaves setup mode then.
	if s.profileSync == nil || s.setupMode.Load() {
		return profileApply{}
	}
	return profileApply{live: true, res: s.profileSync(cfg)}
}

// restartRequired: the change to name only applies after a daemon restart.
func (p profileApply) restartRequired(name string) bool {
	return !p.live || slices.Contains(p.res.Failed, name)
}

// reconnectRequired: name's tunnel is up and keeps its old settings until
// the user reconnects it.
func (p profileApply) reconnectRequired(name string) bool {
	return p.live && (slices.Contains(p.res.Reconnect, name) || slices.Contains(p.res.Kept, name))
}

// tunnelSettingsChanged reports whether an edit touches what a running
// tunnel was brought up with. Priority, failover and the kill switch
// override are read when used, so they never need a reconnect.
func tunnelSettingsChanged(before, after config.VPNConfig) bool {
	strip := func(v config.VPNConfig) config.VPNConfig {
		v.Priority = 0
		v.Failover = nil
		v.KillSwitch = nil
		return v
	}
	return !reflect.DeepEqual(strip(before), strip(after))
}

// profileChangeBody is the JSON answer of a VPN profile change.
func profileChangeBody(status, name string, p profileApply) map[string]any {
	return map[string]any{
		"status":             status,
		"profile":            name,
		"restart_required":   p.restartRequired(name),
		"reconnect_required": p.reconnectRequired(name),
	}
}
