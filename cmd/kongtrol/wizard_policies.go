package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Routing policies ──────────────────────────────────────────────────────────

// collectPoliciesHuh drives the routing-policy step of the wizard. It returns
// the names of policies added or replaced this run (for the review panel) and
// errWizardCancelled if the user aborted.
func collectPoliciesHuh(lang i18n.Lang, doc *yaml.Node, existing *config.Config, profileNames map[string]bool) ([]string, error) {
	StepHeader(2, 4, i18n.T(lang, "section.policies"))

	if existing != nil && len(existing.Policies) > 0 {
		fmt.Println(tuiInfo(i18n.T(lang, "policy.existing")))
		for _, p := range existing.Policies {
			fmt.Printf("    %s  %s → %s\n",
				styleInfo.Render("·"),
				styleBright.Render(p.Name),
				styleWarn.Render(p.Via))
		}
		fmt.Println()
	}

	// Build profile options for "via" selector.
	var profileList []string
	for name := range profileNames {
		profileList = append(profileList, name)
	}
	sort.Strings(profileList)

	if len(profileList) == 0 {
		fmt.Println(styleDim.Render("    " + i18n.T(lang, "policy.no_profiles")))
		return nil, nil
	}

	viaOpts := make([]huh.Option[string], len(profileList))
	for i, n := range profileList {
		viaOpts[i] = huh.NewOption(n, n)
	}

	policiesNode := mappingKey(doc, "policies")
	if policiesNode == nil {
		policiesNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		doc.Content = append(doc.Content, scalarNode("policies"), policiesNode)
	}

	knownPolicies := make(map[string]bool)
	if existing != nil {
		for _, p := range existing.Policies {
			knownPolicies[p.Name] = true
		}
	}

	var added []string
	addedCount := 0
	for {
		defAdd := addedCount == 0 && len(profileList) > 0
		var addNew bool
		if defAdd {
			addNew = true
		}
		if err := runForm(newForm(huh.NewGroup(
			huh.NewConfirm().
				Title(i18n.T(lang, "policy.add_new")).
				Value(&addNew),
		))); err != nil {
			return added, err
		}
		if !addNew {
			break
		}

		var (
			policyName string
			via        string = profileList[0]
			domainsRaw string
			ipsRaw     string
		)

		if err := runForm(newForm(
			huh.NewGroup(
				huh.NewInput().
					Title(i18n.T(lang, "policy.name")).
					Description(styleDim.Render("e.g. work-internal, saas-apps")).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s", i18n.T(lang, "policy.name_empty"))
						}
						return nil
					}).
					Value(&policyName),
				huh.NewSelect[string]().
					Title(i18n.T(lang, "policy.via")).
					Options(viaOpts...).
					Value(&via),
			),
			huh.NewGroup(
				huh.NewNote().
					Title(i18n.T(lang, "policy.domains_hint")).
					Description(styleDim.Render("e.g. internal.company.com, *.corp")),
				huh.NewInput().
					Title(i18n.T(lang, "policy.domain_prompt")).
					Description(styleDim.Render("Comma-separated, leave empty to skip")).
					Value(&domainsRaw),
				huh.NewNote().
					Title(i18n.T(lang, "policy.ips_hint")).
					Description(styleDim.Render("e.g. 10.0.0.0/8, 192.168.1.0/24")),
				huh.NewInput().
					Title(i18n.T(lang, "policy.ip_prompt")).
					Description(styleDim.Render("Comma-separated CIDR, leave empty to skip")).
					Value(&ipsRaw),
			),
		)); err != nil {
			return added, err
		}

		policyName = strings.TrimSpace(policyName)
		if policyName == "" {
			continue
		}

		var domains []string
		for d := range strings.SplitSeq(domainsRaw, ",") {
			if v := strings.TrimSpace(d); v != "" {
				domains = append(domains, v)
			}
		}
		var ipRanges []string
		for ip := range strings.SplitSeq(ipsRaw, ",") {
			if v := strings.TrimSpace(ip); v != "" {
				ipRanges = append(ipRanges, v)
			}
		}

		if len(domains) == 0 && len(ipRanges) == 0 {
			fmt.Println(tuiWarn(i18n.T(lang, "policy.empty_match")))
			continue
		}

		if knownPolicies[policyName] {
			fmt.Println(tuiWarn(i18n.F(lang, "policy.already_exists", policyName)))
			var replace bool
			if err := runForm(newForm(huh.NewGroup(
				huh.NewConfirm().Title(i18n.T(lang, "policy.replace_confirm")).Value(&replace),
			))); err != nil {
				return added, err
			}
			if !replace {
				fmt.Println(styleDim.Render(i18n.T(lang, "policy.replace_skipped")))
				continue
			}
			removePolicyByName(policiesNode, policyName)
		}

		policiesNode.Content = append(policiesNode.Content, policyNode(policyName, via, domains, ipRanges))
		knownPolicies[policyName] = true
		added = append(added, policyName)
		addedCount++
	}

	fmt.Println()
	fmt.Println(styleDim.Render("    " + i18n.T(lang, "policy.yaml_hint")))
	return added, nil
}

// policyNode builds a YAML mapping node for a single policy rule.
func policyNode(name, via string, domains, ipRanges []string) *yaml.Node {
	matchNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(domains) > 0 {
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, d := range domains {
			seq.Content = append(seq.Content, scalarNode(d))
		}
		matchNode.Content = append(matchNode.Content, scalarNode("domains"), seq)
	}
	if len(ipRanges) > 0 {
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, ip := range ipRanges {
			seq.Content = append(seq.Content, scalarNode(ip))
		}
		matchNode.Content = append(matchNode.Content, scalarNode("ip_ranges"), seq)
	}
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	n.Content = append(n.Content,
		scalarNode("name"), scalarNode(name),
		scalarNode("match"), matchNode,
		scalarNode("via"), scalarNode(via),
	)
	return n
}

func removePolicyByName(seq *yaml.Node, name string) {
	if seq == nil {
		return
	}
	for i, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(item.Content); j += 2 {
			if item.Content[j].Value == "name" && item.Content[j+1].Value == name {
				seq.Content = append(seq.Content[:i], seq.Content[i+1:]...)
				return
			}
		}
	}
}
