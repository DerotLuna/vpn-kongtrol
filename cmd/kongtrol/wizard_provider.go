package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/nordvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
)

// ── Provider apps (attach mode) ──────────────────────────────────────────────
//
// forticlient, protonvpn, nordvpn and surfshark work with the provider's
// official app on every OS (see CLAUDE.md "Provider adapters vs direct
// configs"). The wizard only asks the fields valid for each provider and
// writes only answers that differ from the defaults.

// providerAppNames is the official app name shown in the wizard's messages.
var providerAppNames = map[string]string{
	"forticlient": "FortiClient",
	"protonvpn":   "Proton VPN",
	"nordvpn":     "NordVPN",
	"surfshark":   "Surfshark",
	"mullvad":     "Mullvad VPN",
	"pia":         "Private Internet Access",
	"expressvpn":  "ExpressVPN",
	"windscribe":  "Windscribe",
	"cyberghost":  "CyberGhost VPN",
}

// directCapable lists the consumer providers that also offer downloadable
// WireGuard / OpenVPN files (the direct way, no app).
var directCapable = map[string]bool{"protonvpn": true, "nordvpn": true, "surfshark": true, "mullvad": true, "pia": true, "expressvpn": true, "windscribe": true, "cyberghost": true}

// providerAnswers are the attach-mode answers for one provider profile.
type providerAnswers struct {
	AutoConnect                     bool
	Country, City, Group, Server    string
	Interface, AppPath, WaitTimeout string
}

func defaultProviderAnswers() providerAnswers { return providerAnswers{AutoConnect: true} }

// providerCanSelect reports whether Kongtrol can pick a location for the
// provider (validateProvider rejects location fields otherwise).
func providerCanSelect(adapterType string) bool {
	return adapterType == "nordvpn" || adapterType == "protonvpn" || adapterType == "mullvad" || config.ProviderExactLocation(adapterType)
}

// providerAutoAvailable reports whether Kongtrol can connect the provider on
// its own on this OS (CLI or, for FortiClient, GUI automation).
func providerAutoAvailable(adapterType string) bool {
	if desktop, ok := appattach.DesktopDefinition(adapterType); ok {
		return len(desktop.CLI) > 0
	}
	switch adapterType {
	case "forticlient":
		return true
	case "protonvpn":
		return !errors.Is(protonvpn.CheckPlatform(), protonvpn.ErrUnsupportedOS)
	case "nordvpn":
		return !errors.Is(nordvpn.CheckPlatform(), nordvpn.ErrUnsupportedOS)
	}
	return false
}

// providerFields returns the YAML pairs for a, keeping only the fields valid
// for adapterType that differ from the defaults (auto_connect true, wait
// 5m, nothing else set) so the written YAML stays minimal.
func providerFields(adapterType string, a providerAnswers) [][2]string {
	var out [][2]string
	add := func(k, v string) {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, [2]string{k, v})
		}
	}
	if !a.AutoConnect && adapterType != "surfshark" {
		out = append(out, [2]string{"auto_connect", "false"})
	}
	if providerCanSelect(adapterType) {
		if strings.TrimSpace(a.Server) != "" {
			add("server", a.Server)
		} else if !config.ProviderExactLocation(adapterType) {
			add("country", a.Country)
			add("city", a.City)
			if adapterType == "nordvpn" {
				add("group", a.Group)
			}
		}
	}
	add("interface", a.Interface)
	add("app_path", a.AppPath)
	if wt := strings.TrimSpace(a.WaitTimeout); wt != "" {
		if d, err := time.ParseDuration(wt); err != nil || d != appattach.DefaultWaitTimeout {
			add("wait_timeout", wt)
		}
	}
	return out
}

// validateWaitTimeout accepts empty or a Go duration of at least 10s.
func validateWaitTimeout(lang i18n.Lang) func(string) error {
	return func(s string) error {
		if s = strings.TrimSpace(s); s == "" {
			return nil
		}
		if d, err := time.ParseDuration(s); err != nil || d < 10*time.Second {
			return errors.New(i18n.T(lang, "err.wait_timeout"))
		}
		return nil
	}
}

// selectProviderMode asks app vs direct for providers that offer
// downloadable files. It returns the adapter type to configure: the
// provider itself, or wireguard / openvpn for the direct way.
func selectProviderMode(lang i18n.Lang, adapterType string) (string, error) {
	if !directCapable[adapterType] {
		return adapterType, nil
	}
	mode := "app"
	if err := runForm(newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(i18n.T(lang, "collect.provider_mode")).
			Description(styleDim.Render(i18n.T(lang, "hint.provider_mode"))).
			Options(
				huh.NewOption(i18n.T(lang, "provider.mode.app"), "app"),
				huh.NewOption(i18n.T(lang, "provider.mode.direct"), "direct"),
			).
			Value(&mode),
	))); err != nil {
		return "", err
	}
	if mode == "app" {
		return adapterType, nil
	}
	fmt.Println(tuiInfo(i18n.T(lang, "wizard.direct.hint."+adapterType)))
	direct := "wireguard"
	switch adapterType {
	case "mullvad":
		return "wireguard", nil
	case "pia", "expressvpn", "cyberghost":
		return "openvpn", nil
	case "nordvpn":
		direct = "openvpn"
	}
	if err := runForm(newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(i18n.T(lang, "collect.direct_type")).
			Options(
				huh.NewOption("WireGuard (.conf)", "wireguard"),
				huh.NewOption("OpenVPN (.ovpn)", "openvpn"),
			).
			Value(&direct),
	))); err != nil {
		return "", err
	}
	return direct, nil
}

// collectProviderFields asks the attach-mode fields of a provider profile
// and appends the non-default ones to fields.
func collectProviderFields(lang i18n.Lang, adapterType string, fields *[][2]string) error {
	t := func(key string) string { return i18n.T(lang, key) }
	app := providerAppNames[adapterType]
	auto := providerAutoAvailable(adapterType)

	fmt.Println(tuiInfo(i18n.F(lang, "wizard.provider.intro", app)))
	if adapterType != "forticlient" {
		fmt.Println(tuiInfo(t("wizard.provider.login_hint")))
	}

	a := defaultProviderAnswers()
	var groups []*huh.Group
	if auto {
		groups = append(groups, huh.NewGroup(
			huh.NewConfirm().Title(t("collect.auto_connect")).
				Description(styleDim.Render(t("hint.auto_connect"))).
				Value(&a.AutoConnect),
		))
		if providerCanSelect(adapterType) {
			var inputs []huh.Field
			if !config.ProviderExactLocation(adapterType) {
				inputs = append(inputs, huh.NewInput().Title(t("collect.country")).
					Description(styleDim.Render(t("hint.country"))).Value(&a.Country),
					huh.NewInput().Title(t("collect.city")).
						Description(styleDim.Render(t("hint.city"))).Value(&a.City))
			}
			if adapterType == "nordvpn" {
				inputs = append(inputs, huh.NewInput().Title(t("collect.nord_group")).
					Description(styleDim.Render(t("hint.nord_group"))).Value(&a.Group))
			}
			label, hint := "collect.server", "hint.server"
			if config.ProviderExactLocation(adapterType) {
				label, hint = "collect.provider_region", "hint.provider_region"
			}
			inputs = append(inputs, huh.NewInput().Title(t(label)).
				Description(styleDim.Render(t(hint))).
				Validate(func(s string) error {
					if strings.TrimSpace(s) != "" && strings.TrimSpace(a.Country+a.City+a.Group) != "" {
						return errors.New(t("err.server_combo"))
					}
					return nil
				}).
				Value(&a.Server))
			groups = append(groups, huh.NewGroup(inputs...))
		}
	}
	var advanced bool
	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().Title(t("collect.provider_advanced")).Value(&advanced),
	))
	if err := runForm(newForm(groups...)); err != nil {
		return err
	}
	if advanced {
		if err := runForm(newForm(huh.NewGroup(
			huh.NewInput().Title(t("collect.interface")).
				Description(styleDim.Render(t("hint.interface"))).Value(&a.Interface),
			huh.NewInput().Title(t("collect.app_path")).
				Description(styleDim.Render(t("hint.app_path"))).
				Validate(validateExistingPath(lang)).Value(&a.AppPath),
			huh.NewInput().Title(t("collect.wait_timeout")).
				Description(styleDim.Render(t("hint.wait_timeout"))).
				Placeholder("5m").
				Validate(validateWaitTimeout(lang)).Value(&a.WaitTimeout),
		))); err != nil {
			return err
		}
	}

	if auto && a.AutoConnect {
		fmt.Println(tuiInfo(i18n.F(lang, "wizard.provider.auto_info", app)))
	} else {
		fmt.Println(tuiInfo(i18n.F(lang, "wizard.provider.manual_info", app)))
	}
	*fields = append(*fields, providerFields(adapterType, a)...)
	return nil
}
