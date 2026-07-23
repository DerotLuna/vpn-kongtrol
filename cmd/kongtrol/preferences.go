package main

import (
	"crypto/rand"
	"encoding/base64"
	"slices"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// auditHMACKeychainProfile/Key locate the audit log's signing key in the OS
// keychain via config.GetCredential/SetCredential (internal/config/keychain.go)
// — the same mechanism used for VPN profile passwords, keyed under a
// non-profile name so it can't collide with a real VPN profile.
const (
	auditHMACKeychainProfile = "_system"
	auditHMACKeychainKey     = "audit_hmac_key"
)

// loadOrCreateAuditHMACKey returns the persistent key used to sign audit log
// entries, generating and storing a new 32-byte key in the OS keychain on
// first use. Without a real key, security.NewAuditLogger silently produces
// unsigned entries even when security.audit_log.sign is true.
func loadOrCreateAuditHMACKey() []byte {
	if existing, err := config.GetCredential(auditHMACKeychainProfile, auditHMACKeychainKey); err == nil && existing != "" {
		if decoded, err := base64.StdEncoding.DecodeString(existing); err == nil && len(decoded) > 0 {
			return decoded
		}
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	if err := config.SetCredential(auditHMACKeychainProfile, auditHMACKeychainKey, encoded); err != nil {
		// Keychain write failed (e.g. no keychain backend available) — still
		// sign this run's entries with the freshly generated key rather than
		// falling back to unsigned; just note it won't survive a restart.
		return raw
	}
	return raw
}

type preferences = config.Preferences

func loadPreferences() (*preferences, error) {
	return config.LoadPreferences("")
}

func savePreferences(p *preferences) error {
	return config.SavePreferences("", p)
}

// applyDashboardPreferences overrides cfg.Monitor.Dashboard with any
// machine-local preference set via `kongtrol config dashboard`. Called right
// after config.Load in loadConfig(), before anything reads the dashboard
// bind/port.
func applyDashboardPreferences(cfg *config.Config) {
	p, err := loadPreferences()
	if err != nil {
		return
	}
	if p.DashboardPort > 0 {
		cfg.Monitor.Dashboard.Port = p.DashboardPort
	}
	if strings.TrimSpace(p.DashboardBind) != "" {
		cfg.Monitor.Dashboard.Bind = p.DashboardBind
	}
}

func addFavorite(name string) error {
	p, err := loadPreferences()
	if err != nil {
		return err
	}
	if !slices.Contains(p.Favorites, name) {
		p.Favorites = append(p.Favorites, name)
	}
	return savePreferences(p)
}

func removeFavorite(name string) error {
	p, err := loadPreferences()
	if err != nil {
		return err
	}
	out := make([]string, 0, len(p.Favorites))
	for _, n := range p.Favorites {
		if n != name {
			out = append(out, n)
		}
	}
	p.Favorites = out
	return savePreferences(p)
}
