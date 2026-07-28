package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Profile collection ────────────────────────────────────────────────────────

// collectProfileHuh drives the "add a VPN profile" flow: name + adapter type,
// adapter-specific fields, an optional binary-path override, and priority.
// Returns errWizardCancelled if the user aborts at any point.
func collectProfileHuh(lang i18n.Lang, detected []detectedVPN, _ map[string]bool) (string, *yaml.Node, error) {
	t := func(key string) string { return i18n.T(lang, key) }

	SectionHeader(t("section.new_profile"))

	// Build adapter options — detected ones first, with a note.
	allAdapters := []string{"forticlient", "openvpn", "protonvpn", "ciscoanyconnect", "wireguard", "globalprotect", "tailscale", "cloudflarewarp"}
	detectedKeys := detectedAdapterKeys(detected)

	var adapterOpts []huh.Option[string]
	for _, key := range allAdapters {
		if detectedKeys[key] {
			adapterOpts = append(adapterOpts, huh.NewOption(key+" ✓", key))
		}
	}
	for _, key := range allAdapters {
		if !detectedKeys[key] {
			adapterOpts = append(adapterOpts, huh.NewOption(key, key))
		}
	}

	var (
		name        string
		adapterType string
	)
	defAdapter := "openvpn"
	if len(adapterOpts) > 0 {
		// First option is the first detected (or first in list).
		defAdapter = allAdapters[0]
		for _, key := range allAdapters {
			if detectedKeys[key] {
				defAdapter = key
				break
			}
		}
	}
	adapterType = defAdapter

	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewInput().
				Title(t("collect.profile_name")).
				Description(styleDim.Render("e.g. work-vpn, office-forti")).
				Validate(validateProfileName(lang)).
				Value(&name),
			huh.NewSelect[string]().
				Title(t("collect.type")).
				Description(styleDim.Render("✓ = detected on this machine")).
				Options(adapterOpts...).
				Value(&adapterType),
		),
	)); err != nil {
		return "", nil, err
	}

	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
	if name == "" {
		return "", nil, fmt.Errorf("%s", t("collect.profile_name_empty"))
	}

	// Type-specific fields
	fields := [][2]string{{"type", adapterType}}
	auth := config.AuthConfig{}

	switch adapterType {
	case "forticlient":
		if err := collectFortiFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "openvpn":
		if err := collectOpenVPNFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "protonvpn":
		if err := collectProtonFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "ciscoanyconnect":
		if err := collectCiscoFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "wireguard":
		if err := collectWireGuardFields(lang, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "globalprotect":
		if err := collectGlobalProtectFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "tailscale":
		if err := collectTailscaleFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "cloudflarewarp":
		fmt.Println(tuiInfo(i18n.T(lang, "collect.warp_info1")))
		fmt.Println(styleDim.Render("  " + i18n.T(lang, "collect.warp_info2")))
		auth.Method = "credentials"

	default:
		return "", nil, fmt.Errorf("%s", i18n.F(lang, "collect.unknown_adapter", adapterType))
	}

	// Binary path override (when not auto-detected) + priority — one
	// multi-group form so Shift+Tab moves between them instead of committing
	// each in its own isolated prompt.
	needsBinaryPath := adapterType != "cloudflarewarp" && !detectedKeys[adapterType]
	if needsBinaryPath {
		fmt.Println(tuiWarn(i18n.T(lang, "collect.not_detected")))
	}

	var (
		binPath     string
		priorityStr string = "10"
	)
	groups := make([]*huh.Group, 0, 2)
	if needsBinaryPath {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title(i18n.T(lang, "collect.binary_path")).
				Description(styleDim.Render("Leave empty to skip")).
				Value(&binPath),
		))
	}
	groups = append(groups, huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, "collect.priority")).
			Placeholder("10").
			Value(&priorityStr),
	))
	if err := runForm(newForm(groups...)); err != nil {
		return "", nil, err
	}
	if binPath != "" {
		fields = append(fields, [2]string{"binary_path", binPath})
	}
	if priorityStr == "" {
		priorityStr = "10"
	}
	fields = append(fields, [2]string{"priority", priorityStr})

	// Build YAML node
	authFields := [][2]string{{"method", auth.Method}}
	if auth.Cert != "" {
		authFields = append(authFields, [2]string{"cert", auth.Cert})
	}
	if auth.Key != "" {
		authFields = append(authFields, [2]string{"key", auth.Key})
	}
	if auth.Username != "" {
		authFields = append(authFields, [2]string{"username", auth.Username})
	}
	if auth.PasswordKeychain != "" {
		authFields = append(authFields, [2]string{"password_keychain", auth.PasswordKeychain})
	}

	node := mapNode(fields)
	node.Content = append(node.Content, scalarNode("auth"), mapNode(authFields))

	if auth.PasswordKeychain != "" {
		if err := collectCredentialsHuh(lang, name, adapterType, auth); err != nil {
			if errors.Is(err, errWizardCancelled) {
				return "", nil, err
			}
			fmt.Println(tuiWarn(i18n.F(lang, "collect.password_warn", err)))
		}
	}

	return name, node, nil
}
