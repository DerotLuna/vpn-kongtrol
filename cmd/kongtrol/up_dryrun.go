package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type dryRunCheck struct {
	Level  string `json:"level"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

type dryRunProfile struct {
	Name   string        `json:"name"`
	Type   string        `json:"type"`
	Checks []dryRunCheck `json:"checks"`
}

type dryRunReport struct {
	Mode     string          `json:"mode"`
	Profiles []dryRunProfile `json:"profiles"`
	Failures int             `json:"failures"`
	Warnings int             `json:"warnings"`
}

func runUpDryRun(targets []string) error {
	report := dryRunReport{Mode: "dry-run", Profiles: make([]dryRunProfile, 0, len(targets))}
	addGlobalInfo := func(detail string) {
		report.Profiles = append(report.Profiles, dryRunProfile{
			Name: "__global__",
			Type: "meta",
			Checks: []dryRunCheck{
				{Level: "info", Name: "security", Detail: detail},
			},
		})
	}

	if !outputJSON {
		fmt.Println(tuiInfo(styleBright.Render(ct("cli.dry_run.title")) + "  " + styleDim.Render(ct("cli.dry_run.subtitle"))))
	}

	for _, name := range targets {
		profile := dryRunProfile{Name: name}
		vpnCfg, ok := cfg.VPNs[name]
		if !ok {
			report.Failures++
			profile.Checks = append(profile.Checks, dryRunCheck{Level: "fail", Name: "profile", Detail: ct("cli.dry_run.profile_missing")})
			report.Profiles = append(report.Profiles, profile)
			continue
		}
		profile.Type = vpnCfg.Type

		adapter, ok := adapters[name]
		if !ok {
			report.Failures++
			profile.Checks = append(profile.Checks, dryRunCheck{Level: "fail", Name: "adapter", Detail: ct("cli.dry_run.adapter_missing")})
			report.Profiles = append(report.Profiles, profile)
			continue
		}

		status := adapter.Status().Normalize()
		if status == "" {
			status = vpn.StatusDisconnected
		}
		profile.Checks = append(profile.Checks, dryRunCheck{
			Level:  "pass",
			Name:   "adapter_status",
			Detail: fmt.Sprintf(ct("cli.dry_run.adapter_status"), status),
		})

		if vpnCfg.ConfigFile != "" {
			if _, err := os.Stat(vpnCfg.ConfigFile); err != nil {
				report.Failures++
				profile.Checks = append(profile.Checks, dryRunCheck{
					Level:  "fail",
					Name:   "config_file",
					Detail: fmt.Sprintf(ct("cli.dry_run.config_missing"), vpnCfg.ConfigFile),
				})
			} else {
				profile.Checks = append(profile.Checks, dryRunCheck{
					Level:  "pass",
					Name:   "config_file",
					Detail: fmt.Sprintf(ct("cli.dry_run.config_ok"), vpnCfg.ConfigFile),
				})
			}
		}

		checkKeychain := func(key, credName string) {
			if key == "" {
				return
			}
			if _, err := config.GetCredential(name, credName); err != nil {
				report.Failures++
				profile.Checks = append(profile.Checks, dryRunCheck{
					Level:  "fail",
					Name:   "keychain_" + credName,
					Detail: fmt.Sprintf(ct("cli.dry_run.keychain_missing"), credName),
				})
				return
			}
			profile.Checks = append(profile.Checks, dryRunCheck{
				Level:  "pass",
				Name:   "keychain_" + credName,
				Detail: fmt.Sprintf(ct("cli.dry_run.keychain_ok"), credName),
			})
		}
		checkKeychain(vpnCfg.Auth.UsernameKeychain, "username")
		checkKeychain(vpnCfg.Auth.PasswordKeychain, "password")

		if vpnCfg.Type == "wireguard" && vpnCfg.ConfigFile != "" {
			if cidrs := policyAllowedIPs(name); len(cidrs) > 0 {
				sample := strings.Join(cidrs, ", ")
				if len(sample) > 96 {
					sample = sample[:95] + "…"
				}
				profile.Checks = append(profile.Checks, dryRunCheck{
					Level:  "info",
					Name:   "wireguard_allowed_ips",
					Detail: fmt.Sprintf(ct("cli.dry_run.wg_split"), len(cidrs), sample),
				})
			} else {
				profile.Checks = append(profile.Checks, dryRunCheck{
					Level:  "info",
					Name:   "wireguard_allowed_ips",
					Detail: ct("cli.dry_run.wg_full"),
				})
			}
		}

		if cfg.Security.DNSGuard.Enabled && vpnCfg.Type != "wireguard" {
			fallback := cfg.Security.DNSGuard.FallbackDNS
			if fallback == "" {
				fallback = ct("cli.dry_run.none")
			}
			profile.Checks = append(profile.Checks, dryRunCheck{
				Level:  "info",
				Name:   "dns_guard",
				Detail: fmt.Sprintf(ct("cli.dry_run.dns_guard"), fallback),
			})
		}

		report.Profiles = append(report.Profiles, profile)
	}

	if cfg.Security.KillSwitch.Enabled {
		addGlobalInfo(ct("cli.dry_run.kill_switch_enabled"))
	} else {
		addGlobalInfo(ct("cli.dry_run.kill_switch_disabled"))
	}

	if outputJSON {
		if err := emitJSON(report); err != nil {
			return err
		}
	} else {
		printDryRunReport(report)
	}

	if report.Failures > 0 {
		return fmt.Errorf(ct("cli.dry_run.failed"), report.Failures)
	}
	return nil
}

func printDryRunReport(report dryRunReport) {
	for _, profile := range report.Profiles {
		if profile.Name == "__global__" {
			for _, check := range profile.Checks {
				fmt.Println("  " + styleInfo.Render(">") + " " + styleDim.Render(check.Detail))
			}
			continue
		}

		fmt.Println()
		fmt.Println("  " + styleBright.Render(profile.Name) + "  " + styleDim.Render("("+profile.Type+")"))

		for _, check := range profile.Checks {
			switch check.Level {
			case "pass":
				fmt.Println("    " + styleOK.Render(sym("✓", "[OK]")) + " " + styleDim.Render(check.Detail))
			case "fail":
				fmt.Println("    " + styleErr.Render(sym("✗", "[ERR]")) + " " + styleDim.Render(check.Detail))
			default:
				fmt.Println("    " + styleInfo.Render(">") + " " + styleDim.Render(check.Detail))
			}
		}
	}

	fmt.Println()
	if report.Failures == 0 {
		fmt.Println(tuiOK(styleBright.Render(ct("cli.dry_run.passed")) + "  " + styleDim.Render(ct("cli.dry_run.ready"))))
	}
}
