package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Edit menu ──────────────────────────────────────────────────────────────────

// runEditMenu drives `init` against an already-configured kongtrol.yaml: an
// action menu (add profile, refresh credentials, policies, security, save)
// repeated until the user saves or exits, instead of re-running every setup
// step in a fixed order. Returns errWizardCancelled if the user aborts the
// menu itself; individual actions report their own cancellation.
func runEditMenu(lang i18n.Lang, doc *yaml.Node, existing *config.Config, detected []detectedVPN, outPath, home string, tf func(string, ...any) string, cancelled func() error) error {
	t := func(key string) string { return i18n.T(lang, key) }

	knownProfiles := make(map[string]bool, len(existing.VPNs))
	for name := range existing.VPNs {
		knownProfiles[name] = true
	}
	var addedProfiles []profileSummary
	var addedPolicies []string
	sec := securitySummary{
		killSwitch: existing.Security.KillSwitch.Enabled,
		dnsGuard:   existing.Security.DNSGuard.Enabled,
		auditLog:   existing.Security.AuditLog.Sign,
		monitor:    existing.Monitor.Enabled,
	}

	for {
		clearScreen()
		printExistingProfiles(tf, outPath, existing)
		var action string
		if err := runForm(newForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(t("menu.title")).
					Options(
						huh.NewOption(t("menu.add_profile"), "add_profile"),
						huh.NewOption(t("menu.refresh_creds"), "refresh_creds"),
						huh.NewOption(t("menu.policies"), "policies"),
						huh.NewOption(t("menu.security"), "security"),
						huh.NewOption(t("menu.review_save"), "review_save"),
						huh.NewOption(t("menu.exit"), "exit"),
					).
					Value(&action),
			),
		)); err != nil {
			return cancelled()
		}

		// Ctrl+C/Esc inside an action backs out to the menu, not out of the
		// whole wizard — only "Exit" (or Ctrl+C at the menu select itself,
		// handled above) actually leaves. backToMenu reports that; real
		// (non-cancellation) errors get their own message.
		backToMenu := func(err error) {
			if errors.Is(err, errWizardCancelled) {
				fmt.Println(styleDim.Render("  " + t("menu.action_cancelled")))
				return
			}
			fmt.Println(tuiErr(err.Error()))
		}

		switch action {
		case "add_profile":
			if err := addProfileFlow(lang, doc, detected, knownProfiles, &addedProfiles); err != nil {
				backToMenu(err)
			}

		case "refresh_creds":
			if err := refreshCredentialsFlow(lang, existing); err != nil {
				backToMenu(err)
			}

		case "policies":
			added, err := collectPoliciesHuh(lang, doc, existing, knownProfiles)
			if err != nil {
				backToMenu(err)
				continue
			}
			addedPolicies = append(addedPolicies, added...)

		case "security":
			newSec, err := collectSecurityHuh(lang, doc, home, sec)
			if err != nil {
				backToMenu(err)
				continue
			}
			sec = newSec

		case "review_save":
			if err := reviewAndWrite(lang, outPath, doc, addedProfiles, addedPolicies, sec); err != nil {
				if errors.Is(err, errWizardCancelled) {
					fmt.Println(styleDim.Render("  " + t("menu.action_cancelled")))
					continue
				}
				return err
			}
			return nil

		case "exit":
			fmt.Println(styleWarn.Render(t("wizard.cancelled")))
			return nil
		}
	}
}

// refreshCredentialsFlow lets the user pick, from a single multi-select over
// every existing profile, which ones to refresh credentials for — instead of
// one confirm per profile. Returns errWizardCancelled if the user aborts.
func refreshCredentialsFlow(lang i18n.Lang, existing *config.Config) error {
	t := func(key string) string { return i18n.T(lang, key) }

	names := make([]string, 0, len(existing.VPNs))
	for name := range existing.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)

	opts := make([]huh.Option[string], len(names))
	for i, name := range names {
		opts[i] = huh.NewOption(name+" ("+existing.VPNs[name].Type+")", name)
	}

	var toRefresh []string
	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(t("profile.refresh_select")).
				Options(opts...).
				Value(&toRefresh),
		),
	)); err != nil {
		return err
	}

	for _, name := range toRefresh {
		vpnCfg := existing.VPNs[name]
		if err := collectCredentialsHuh(lang, name, vpnCfg.Type, vpnCfg.Auth); err != nil {
			if errors.Is(err, errWizardCancelled) {
				return err
			}
			fmt.Println(tuiWarn(err.Error()))
		}
	}
	return nil
}
