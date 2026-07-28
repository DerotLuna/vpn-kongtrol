package main

import (
	"github.com/charmbracelet/huh"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Credentials ───────────────────────────────────────────────────────────────

// collectCredentialsHuh prompts for and stores the secret half of a profile's
// auth (password or Tailscale auth key) directly in the OS keychain. Returns
// errWizardCancelled if the user aborts the prompt.
func collectCredentialsHuh(lang i18n.Lang, profileName, adapterType string, auth config.AuthConfig) error {
	switch adapterType {
	case "cloudflarewarp", "wireguard":
		return nil
	case "tailscale":
		if auth.PasswordKeychain == "" {
			return nil
		}
		var key string
		if err := runForm(newForm(huh.NewGroup(
			huh.NewInput().
				Title(i18n.T(lang, "collect.ts_key")).
				EchoMode(huh.EchoModePassword).
				Value(&key),
		))); err != nil {
			return err
		}
		if key != "" {
			return config.SetCredential(profileName, "password", key)
		}
		return nil
	default:
		if auth.PasswordKeychain == "" {
			return nil
		}
		var pwd string
		title := i18n.F(lang, "collect.password", styleBright.Render(profileName))
		if err := runForm(newForm(huh.NewGroup(
			huh.NewInput().
				Title(title).
				EchoMode(huh.EchoModePassword).
				Value(&pwd),
		))); err != nil {
			return err
		}
		if pwd != "" {
			return config.SetCredential(profileName, "password", pwd)
		}
		return nil
	}
}
