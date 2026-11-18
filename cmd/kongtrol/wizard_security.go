package main

import (
	"path/filepath"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Security ───────────────────────────────────────────────────────────────────

// collectSecurityHuh drives the security toggles step (kill switch, DNS
// guard, audit log, dashboard/monitor). current supplies the starting values
// (the existing config's settings when editing, or all-on defaults on first
// setup) so re-visiting this step from the edit menu doesn't reset choices
// made earlier in the session. Returns errWizardCancelled if the user aborts.
func collectSecurityHuh(lang i18n.Lang, doc *yaml.Node, home string, current securitySummary) (securitySummary, error) {
	t := func(key string) string { return i18n.T(lang, key) }

	StepHeader(3, 4, t("section.security"))

	secNode := mappingKey(doc, "security")
	if secNode == nil {
		secNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Content = append(doc.Content, scalarNode("security"), secNode)
	}

	enableKS := current.killSwitch
	enableDNS := current.dnsGuard
	enableAudit := current.auditLog
	enableMonitor := current.monitor
	enablePublicIP := current.publicIP
	enableUpdateCheck := current.updateCheck

	if err := runForm(newForm(
		huh.NewGroup(
			huh.NewNote().
				Title(t("section.security")).
				Description(styleDim.Render(t("security.note"))),
			huh.NewConfirm().
				Title(t("security.kill_switch")).
				Description(styleDim.Render(t("hint.killswitch"))).
				Value(&enableKS),
			huh.NewConfirm().
				Title(t("security.dns_guard")).
				Description(styleDim.Render(t("hint.dnsguard"))).
				Value(&enableDNS),
			huh.NewConfirm().
				Title(t("security.audit_log")).
				Description(styleDim.Render(t("hint.auditlog"))).
				Value(&enableAudit),
			huh.NewConfirm().
				Title(t("monitor.dashboard")).
				Description(styleDim.Render(t("hint.dashboard"))).
				Value(&enableMonitor),
			huh.NewConfirm().
				Title(t("security.public_ip")).
				Description(styleDim.Render(i18n.F(lang, "cli.public_ip.notice", config.EndpointHost(config.DefaultPublicIPEndpoint)))).
				Value(&enablePublicIP),
			huh.NewConfirm().
				Title(t("update.wizard.title")).
				Description(styleDim.Render(t("update.wizard.desc"))).
				Value(&enableUpdateCheck),
		),
	)); err != nil {
		return securitySummary{}, err
	}

	auditPath := filepath.Join(home, ".kongtrol", "audit.log")
	if enableKS {
		setMapping(secNode, "kill_switch", mapNode([][2]string{
			{"enabled", "true"}, {"mode", "strict"}, {"allow_lan", "true"},
		}))
	}
	if enableDNS {
		setMapping(secNode, "dns_guard", mapNode([][2]string{
			{"enabled", "true"}, {"fallback_dns", "1.1.1.1"},
		}))
	}
	if enableAudit {
		setMapping(secNode, "audit_log", mapNode([][2]string{
			{"path", auditPath}, {"max_size_mb", "100"}, {"sign", "true"},
		}))
	}
	if enableMonitor {
		setMapping(doc, "monitor", mapNode([][2]string{{"enabled", "true"}}))
	}
	// Public IP lookup: opt-in; turning it off also clears the legacy
	// leak_detection.enabled that implied it.
	// Keeps any custom endpoints already in the file.
	if pic := mappingKey(secNode, "public_ip_check"); pic != nil && pic.Kind == yaml.MappingNode {
		setMapping(pic, "enabled", boolNode(enablePublicIP))
	} else if enablePublicIP {
		setMapping(secNode, "public_ip_check", mapNode([][2]string{{"enabled", "true"}}))
	}
	if ld := mappingKey(secNode, "leak_detection"); !enablePublicIP && ld != nil && ld.Kind == yaml.MappingNode {
		setMapping(ld, "enabled", boolNode(false))
	}

	return securitySummary{
		killSwitch: enableKS, dnsGuard: enableDNS, auditLog: enableAudit, monitor: enableMonitor,
		publicIP: enablePublicIP, updateCheck: enableUpdateCheck,
	}, nil
}
