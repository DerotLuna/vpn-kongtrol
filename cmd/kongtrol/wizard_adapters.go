package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Shared prompt helpers ─────────────────────────────────────────────────────
//
// Every adapter collector below is 80% the same three shapes: a validated
// host input, a "username + keychain password" auth pair, and a plain
// single-field prompt. Centralizing them here is what keeps each
// collectXFields function down to just what's actually adapter-specific.

// promptHost shows a single validated host input and returns the trimmed
// value, or errWizardCancelled if the user aborts.
func promptHost(lang i18n.Lang, titleKey, hintKey string) (string, error) {
	var host string
	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, titleKey)).
			Description(styleDim.Render(i18n.T(lang, hintKey))).
			Validate(validateHost(lang)).
			Value(&host),
	))); err != nil {
		return "", err
	}
	return strings.TrimSpace(host), nil
}

// promptCredentialAuth fills auth with the "credentials" method: prompts for
// a username under usernameTitleKey and wires up the keychain slot the
// password will later be stored under (profileName + ".password").
func promptCredentialAuth(lang i18n.Lang, profileName, usernameTitleKey string, auth *config.AuthConfig) error {
	username, err := promptInput(lang, usernameTitleKey, "")
	if err != nil {
		return err
	}
	auth.Method = "credentials"
	auth.Username = username
	auth.PasswordKeychain = profileName + ".password"
	return nil
}

// promptInput shows a single-field huh form and returns the trimmed value.
func promptInput(lang i18n.Lang, key, placeholder string) (string, error) {
	var val string
	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, key)).
			Placeholder(placeholder).
			Value(&val),
	))); err != nil {
		return "", err
	}
	return strings.TrimSpace(val), nil
}

// promptPath is promptInput plus a validator requiring the path (if
// non-empty) to exist on disk.
func promptPath(lang i18n.Lang, key, placeholder string) (string, error) {
	var val string
	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, key)).
			Placeholder(placeholder).
			Validate(validateExistingPath(lang)).
			Value(&val),
	))); err != nil {
		return "", err
	}
	return strings.TrimSpace(val), nil
}

// selectAuthMethod shows an auth method selector and returns the chosen value.
func selectAuthMethod(lang i18n.Lang, def string) (string, error) {
	method := def
	if err := runForm(newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(i18n.T(lang, "collect.auth_method")).
			Options(
				huh.NewOption(i18n.T(lang, "auth.hint.credentials"), "credentials"),
				huh.NewOption(i18n.T(lang, "auth.hint.certificate"), "certificate"),
				huh.NewOption(i18n.T(lang, "auth.hint.cert+creds"), "certificate+credentials"),
			).
			Value(&method),
	))); err != nil {
		return "", err
	}
	return method, nil
}

// ── Type-specific field collectors ───────────────────────────────────────────

func collectFortiFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		host       string
		port       string = "443"
		tunnelName string = "Office"
		version    string = "6"
	)

	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.host")).
			Description(styleDim.Render(i18n.T(lang, "hint.host.forti"))).
			Validate(validateHost(lang)).
			Value(&host),
		huh.NewInput().Title(i18n.T(lang, "collect.port")).
			Placeholder("443").Validate(validatePort(lang)).Value(&port),
		huh.NewInput().Title(i18n.T(lang, "collect.tunnel_name")).
			Placeholder("Office").Value(&tunnelName),
		huh.NewSelect[string]().Title(i18n.T(lang, "collect.forti_ver")).
			Options(
				huh.NewOption("6.x ("+i18n.T(lang, "forti.ver.hint.6")+")", "6"),
				huh.NewOption("7.x ("+i18n.T(lang, "forti.ver.hint.7")+")", "7"),
				huh.NewOption("5.x ("+i18n.T(lang, "forti.ver.hint.5")+")", "5"),
			).Value(&version),
	))); err != nil {
		return err
	}
	if port == "" {
		port = "443"
	}
	if tunnelName == "" {
		tunnelName = "Office"
	}

	*fields = append(*fields,
		[2]string{"host", host},
		[2]string{"port", port},
		[2]string{"tunnel_name", tunnelName},
		[2]string{"version", version},
	)

	if runtime.GOOS == "windows" {
		fmt.Println(styleDim.Render(i18n.T(lang, "hint.auth.forti.win")))
		return promptCredentialAuth(lang, name, "collect.username", auth)
	}

	method, err := selectAuthMethod(lang, "certificate+credentials")
	if err != nil {
		return err
	}
	auth.Method = method
	if strings.Contains(method, "certificate") {
		if auth.Cert, err = promptPath(lang, "collect.cert", ""); err != nil {
			return err
		}
		if auth.Key, err = promptPath(lang, "collect.key", ""); err != nil {
			return err
		}
	}
	if strings.Contains(method, "credentials") {
		if auth.Username, err = promptInput(lang, "collect.username", ""); err != nil {
			return err
		}
		auth.PasswordKeychain = name + ".password"
	}
	return nil
}

func collectOpenVPNFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	ovpnConfig, err := promptPath(lang, "collect.ovpn_config", "")
	if err != nil {
		return err
	}
	*fields = append(*fields, [2]string{"config", ovpnConfig})

	method, err := selectAuthMethod(lang, "certificate")
	if err != nil {
		return err
	}
	auth.Method = method
	if strings.Contains(method, "certificate") {
		if auth.Cert, err = promptPath(lang, "collect.ovpn_cert", ""); err != nil {
			return err
		}
		if auth.Key, err = promptPath(lang, "collect.ovpn_key", ""); err != nil {
			return err
		}
	}
	if strings.Contains(method, "credentials") {
		return promptCredentialAuth(lang, name, "collect.username", auth)
	}
	return nil
}

func collectProtonFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		server   string = "fastest"
		protocol string = "wireguard"
	)
	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.proton_srv")).
			Description(styleDim.Render(i18n.T(lang, "hint.proton_srv"))).
			Placeholder("fastest").Value(&server),
		huh.NewSelect[string]().Title(i18n.T(lang, "collect.proton_proto")).
			Options(
				huh.NewOption("WireGuard — "+i18n.T(lang, "proto.hint.wireguard"), "wireguard"),
				huh.NewOption("OpenVPN — "+i18n.T(lang, "proto.hint.openvpn"), "openvpn"),
			).Value(&protocol),
	))); err != nil {
		return err
	}
	if server == "" {
		server = "fastest"
	}
	*fields = append(*fields, [2]string{"server", server}, [2]string{"protocol", protocol})
	return promptCredentialAuth(lang, name, "collect.proton_user", auth)
}

func collectCiscoFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	host, err := promptHost(lang, "collect.cisco_host", "hint.host.cisco")
	if err != nil {
		return err
	}
	*fields = append(*fields, [2]string{"host", host})
	return promptCredentialAuth(lang, name, "collect.cisco_user", auth)
}

func collectWireGuardFields(lang i18n.Lang, fields *[][2]string, auth *config.AuthConfig) error {
	wgConfig, err := promptPath(lang, "collect.wg_config", "")
	if err != nil {
		return err
	}
	*fields = append(*fields, [2]string{"config", wgConfig})
	auth.Method = "certificate"
	return nil
}

func collectGlobalProtectFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	host, err := promptHost(lang, "collect.gp_host", "hint.host.gp")
	if err != nil {
		return err
	}
	*fields = append(*fields, [2]string{"host", host})
	return promptCredentialAuth(lang, name, "collect.gp_user", auth)
}

func collectTailscaleFields(lang i18n.Lang, name string, fields *[][2]string, auth *config.AuthConfig) error {
	var (
		exitNode string
		useKey   bool
	)
	if err := runForm(newForm(huh.NewGroup(
		huh.NewInput().Title(i18n.T(lang, "collect.ts_exitnode")).
			Description(styleDim.Render("Leave empty for default routing")).
			Value(&exitNode),
		huh.NewConfirm().Title(i18n.T(lang, "collect.ts_usekey")).
			Value(&useKey),
	))); err != nil {
		return err
	}
	if exitNode != "" {
		*fields = append(*fields, [2]string{"host", exitNode})
	}
	auth.Method = "credentials"
	if useKey {
		auth.PasswordKeychain = name + ".authkey"
	}
	return nil
}
