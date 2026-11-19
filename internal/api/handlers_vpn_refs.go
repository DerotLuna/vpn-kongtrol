package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// profileRefs lists where a profile name is used in the config.
type profileRefs struct {
	Policies, Groups, Failover, Rules []string
}

func (r profileRefs) empty() bool {
	return len(r.Policies)+len(r.Groups)+len(r.Failover)+len(r.Rules) == 0
}

// findProfileRefs returns every policy (via), group, other profile's
// failover list and scheduler rule that names profile. Names are compared
// exactly, as config.Validate does.
func findProfileRefs(cfg *config.Config, profile string) profileRefs {
	var r profileRefs
	for _, p := range cfg.Policies {
		if p.Via == profile {
			r.Policies = append(r.Policies, p.Name)
		}
	}
	for g, gc := range cfg.Groups {
		if slices.Contains(gc.Profiles, profile) {
			r.Groups = append(r.Groups, g)
		}
	}
	for n, v := range cfg.VPNs {
		if n != profile && slices.Contains(v.Failover, profile) {
			r.Failover = append(r.Failover, n)
		}
	}
	for _, rule := range cfg.Monitor.Scheduler.Rules {
		if slices.Contains(rule.Profiles, profile) {
			r.Rules = append(r.Rules, rule.Name)
		}
	}
	sort.Strings(r.Groups)
	sort.Strings(r.Failover)
	return r
}

// inUseError is the localized 409 for deleting a referenced profile: every
// list (or "—") so the user can fix all of them at once.
func inUseError(profile string, r profileRefs) error {
	list := func(xs []string) string {
		if len(xs) == 0 {
			return "—"
		}
		return strings.Join(xs, ", ")
	}
	return &config.ValidationError{Key: "api.err.vpn_in_use",
		Args: []any{profile, list(r.Policies), list(r.Groups), list(r.Failover), list(r.Rules)}}
}

// renameProfileRefs rewrites every reference to from so it names to:
// policies, groups, failover lists, scheduler rules and expected IPs. It
// builds fresh slices and maps, so cfg's originals stay untouched (cfg is
// a shallow trial copy).
func renameProfileRefs(cfg *config.Config, from, to string) {
	swap := func(xs []string) []string {
		out := make([]string, len(xs))
		for i, x := range xs {
			if x == from {
				x = to
			}
			out[i] = x
		}
		return out
	}
	pols := make([]config.PolicyRule, len(cfg.Policies))
	for i, p := range cfg.Policies {
		if p.Via == from {
			p.Via = to
		}
		pols[i] = p
	}
	cfg.Policies = pols
	if cfg.Groups != nil {
		groups := make(map[string]config.GroupConfig, len(cfg.Groups))
		for g, gc := range cfg.Groups {
			groups[g] = config.GroupConfig{Profiles: swap(gc.Profiles)}
		}
		cfg.Groups = groups
	}
	vpns := make(map[string]config.VPNConfig, len(cfg.VPNs))
	for n, v := range cfg.VPNs {
		if len(v.Failover) > 0 {
			v.Failover = swap(v.Failover)
		}
		if n == from {
			n = to
		}
		vpns[n] = v
	}
	cfg.VPNs = vpns
	rules := make([]config.ScheduleRule, len(cfg.Monitor.Scheduler.Rules))
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		rule.Profiles = swap(rule.Profiles)
		rules[i] = rule
	}
	cfg.Monitor.Scheduler.Rules = rules
	if ips := cfg.Security.IntegrityCheck.ExpectedIPs; ips != nil {
		m := make(map[string]string, len(ips))
		for n, ip := range ips {
			if n == from {
				n = to
			}
			m[n] = ip
		}
		cfg.Security.IntegrityCheck.ExpectedIPs = m
	}
}

// profileBusy reports whether the running daemon has the profile up, coming
// up or waiting for the user: renaming it then would orphan that tunnel.
func (s *Server) profileBusy(name string) bool {
	s.connectMu.Lock()
	_, connecting := s.connectCancel[name]
	s.connectMu.Unlock()
	if connecting {
		return true
	}
	a, ok := s.adapters[name]
	return ok && a.Status().Normalize() != vpn.StatusDisconnected
}

// renameVPNProfile renames profile from → to and applies req on top, in one
// trial-then-commit step: every reference follows the new name, the
// keychain entries move (copied first, old ones deleted only after the
// config is written; any failure rolls back), and favorites follow.
// Returns the HTTP status for the error.
func (s *Server) renameVPNProfile(cfg *config.Config, cfgPath, from, to string, req configedit.VPNProfile) (int, error) {
	if err := config.ValidateName("vpn", to); err != nil {
		return http.StatusBadRequest, err
	}
	if _, exists := cfg.VPNs[to]; exists {
		return http.StatusConflict, &config.ValidationError{Key: "api.err.vpn_exists", Args: []any{to}}
	}
	if s.profileBusy(from) {
		return http.StatusConflict, &config.ValidationError{Key: "api.err.vpn_rename_busy", Args: []any{from}}
	}

	trial := *cfg
	renameProfileRefs(&trial, from, to)
	vc := trial.VPNs[to]
	// Keychain item names follow the profile name (configedit.SaveVPNProfile).
	moveUser := vc.Auth.UsernameKeychain == from+".username"
	movePass := vc.Auth.PasswordKeychain == from+".password"
	if moveUser {
		vc.Auth.UsernameKeychain = to + ".username"
	}
	if movePass {
		vc.Auth.PasswordKeychain = to + ".password"
	}
	trial.VPNs[to] = vc
	// The body's other fields apply to the renamed profile; configedit.SaveVPNProfile
	// validates the whole trial and stores new credentials under to.
	req.Name = nil

	var moved []string
	rollback := func() error {
		var errs []error
		for _, k := range moved {
			if err := config.DeleteCredential(to, k); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	for _, k := range []struct {
		key  string
		move bool
	}{{"username", moveUser}, {"password", movePass}} {
		if !k.move {
			continue
		}
		v, err := config.GetCredential(from, k.key)
		if err != nil {
			continue // nothing stored under the old name
		}
		if err := config.SetCredential(to, k.key, v); err != nil {
			return http.StatusInternalServerError, errors.Join(fmt.Errorf("move %s credential: %w", k.key, err), rollback())
		}
		moved = append(moved, k.key)
	}

	if err := configedit.SaveVPNProfile(&trial, to, trial.VPNs[to], req, s.saver(cfgPath)); err != nil {
		status := http.StatusBadRequest
		var ve *config.ValidationError
		if !errors.As(err, &ve) {
			status = http.StatusInternalServerError
		}
		return status, errors.Join(err, rollback())
	}
	*cfg = trial
	for _, k := range moved {
		_ = config.DeleteCredential(from, k) // the copy is in place and saved
	}
	// Favorites live in preferences.json ("" = the default path).
	if prefs, err := config.LoadPreferences(s.preferencesPath); err == nil && slices.Contains(prefs.Favorites, from) {
		_, _ = config.UpdatePreferences(s.preferencesPath, func(p *config.Preferences) error {
			for i, f := range p.Favorites {
				if f == from {
					p.Favorites[i] = to
				}
			}
			return nil
		})
	}
	return http.StatusOK, nil
}
