package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// ── Profile collection ────────────────────────────────────────────────────────

// collectProfileHuh drives the "add a VPN profile" flow: name + adapter type,
// adapter-specific fields, an optional binary-path override, and priority.
// Returns errWizardCancelled if the user aborts at any point.
func collectProfileHuh(lang i18n.Lang, detected []detectedVPN, _ map[string]bool) (string, *yaml.Node, error) {
	t := func(key string) string { return i18n.T(lang, key) }

	SectionHeader(t("section.new_profile"))

	// Build adapter options — detected ones first, with a note.
	allAdapters := []string{"forticlient", "openvpn", "protonvpn", "nordvpn", "surfshark", "ciscoanyconnect", "wireguard", "globalprotect", "tailscale", "cloudflarewarp"}
	allAdapters = append(allAdapters, appattach.DesktopTypes()...)
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
				Validate(validateProfileName(lang)).
				Value(&name),
			huh.NewSelect[string]().
				Title(t("collect.type")).
				Description(styleDim.Render(t("hint.type"))).
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

	// Consumer providers: official app or the direct way (wireguard /
	// openvpn with the provider's downloaded file).
	adapterType, err := selectProviderMode(lang, adapterType)
	if err != nil {
		return "", nil, err
	}

	// Type-specific fields
	fields := [][2]string{{"type", adapterType}}
	auth := config.AuthConfig{}

	switch adapterType {
	case "forticlient":
		if err := collectFortiFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}
		if err := collectProviderFields(lang, adapterType, &fields); err != nil {
			return "", nil, err
		}

	case "openvpn":
		if err := collectOpenVPNFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "protonvpn", "nordvpn", "surfshark", "mullvad", "pia", "expressvpn", "windscribe", "cyberghost":
		// Login lives in the official app: no auth block is written
		// (config defaults auth.method for these types).
		if err := collectProviderFields(lang, adapterType, &fields); err != nil {
			return "", nil, err
		}

	case "ciscoanyconnect":
		if err := collectCiscoFields(lang, name, &fields, &auth); err != nil {
			return "", nil, err
		}

	case "wireguard":
		if err := collectWireGuardFields(lang, name, &fields, &auth); err != nil {
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

	// Binary path override + priority — one multi-group form so Shift+Tab
	// moves between them instead of committing each in its own isolated
	// prompt. The detected binary is kept as the prefilled suggestion; when
	// the adapter already finds it on its own, nothing is asked or written.
	detectedPath := detectedBinaryPath(detected, adapterType)
	needsBinaryPath, binPath := binaryPathPlan(adapterType, detectedPath, vpndetect.IsStandard)
	if directCapable[adapterType] {
		// App providers find their app on their own; app_path is in the
		// advanced options.
		needsBinaryPath, binPath = false, ""
	}
	binHint := t("wizard.binary_path.empty_hint")
	if needsBinaryPath {
		if detectedPath == "" {
			fmt.Println(tuiWarn(t("collect.not_detected")))
		} else {
			binHint = t("wizard.binary_path.detected_hint")
		}
	}

	// 10 like the panel's new profiles; emptied = no priority: the rule's
	// position decides (ADR-0108).
	priorityStr := "10"
	groups := make([]*huh.Group, 0, 2)
	if needsBinaryPath {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title(t("collect.binary_path")).
				Description(styleDim.Render(binHint)).
				Value(&binPath),
		))
	}
	groups = append(groups, huh.NewGroup(
		huh.NewInput().
			Title(i18n.T(lang, "collect.priority")).
			Description(styleDim.Render(t("hint.priority"))).
			Placeholder(t("wizard.priority.placeholder")).
			Value(&priorityStr).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return nil
				}
				_, err := parsePriority(name, s)
				return err
			}),
	))
	if err := runForm(newForm(groups...)); err != nil {
		return "", nil, err
	}
	binPath = strings.TrimSpace(binPath)
	if binPath != "" && !vpndetect.IsStandard(adapterType, binPath) {
		fields = append(fields, [2]string{"binary_path", binPath})
	}
	if p, err := parsePriority(name, priorityStr); err == nil && p > 0 {
		fields = append(fields, [2]string{"priority", strconv.Itoa(p)})
	}

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
	if auth.Method != "" {
		node.Content = append(node.Content, scalarNode("auth"), mapNode(authFields))
	}

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
