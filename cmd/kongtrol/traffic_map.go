package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// ── map ──────────────────────────────────────────────────────────────────────

// mapCmd is the deprecated kongtrol map: kongtrol policy list (all) and
// kongtrol policy show <destino> (one) replace it. It keeps its old output
// (and --json shape) for scripts.
var mapCmd = deprecatedCommand("map [target|app:<exe>]", "kongtrol map", "kongtrol policy list · kongtrol policy show", cobra.MaximumNArgs(1), runMap)

func runMap(cmd *cobra.Command, args []string) error {
	if engine == nil {
		return fmt.Errorf("%s", ct("cli.policy.engine_not_loaded"))
	}
	if outputJSON {
		return emitJSON(buildMapReport(args))
	}

	// If a target is given, resolve it and print the result.
	if len(args) > 0 {
		printResolve(args[0])
		fmt.Println()
	}

	printTrafficMap(false)
	return nil
}

type mapRuleJSON struct {
	Name        string   `json:"name"`
	Via         string   `json:"via"`
	Apps        []string `json:"apps,omitempty"`
	Domains     []string `json:"domains,omitempty"`
	IPRanges    []string `json:"ip_ranges,omitempty"`
	ResolvedIPs int      `json:"resolved_ips"`
	// Priority of the rule's profile; 0 = its position decides (ADR-0108).
	Priority int `json:"priority"`
}

type mapReportJSON struct {
	Target *policy.ExplainResult `json:"target,omitempty"`
	Rules  []mapRuleJSON         `json:"rules"`
}

// Map table styles live in theme.go ("Signal Contour").

func printResolve(target string) {
	ex := engine.ExplainTarget(target)
	if ex.Matched {
		fmt.Printf("  %s  %s  →  %s\n",
			styleStatusUp.Render("●"),
			styleMapName.Render(target),
			styleMapVia.Render(ex.Via))
		if ex.RuleName != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.rule_label")) + styleBright.Render(ex.RuleName))
		}
		if ex.Reason != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
		}
	} else {
		fmt.Printf("  %s  %s  →  %s\n",
			styleDim.Render("○"),
			styleBright.Render(target),
			styleDim.Render(ct("cli.policy.default_route")))
		if ex.Reason != "" {
			fmt.Println("     " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
		}
	}
}

// printTrafficMap prints the rules table; withOrder adds the ORDER column
// (the profile's priority or "by position", ADR-0108) that `policy list`
// shows. The deprecated `map` keeps its old columns.
func printTrafficMap(withOrder bool) {
	rules := engine.Rules()
	if len(rules) == 0 {
		fmt.Println("  " + styleDim.Render(ct("cli.map.none")))
		return
	}

	// Get resolved IPs from PolicyResolver if available.
	var resolvedByProfile map[string]int
	if policyResolver != nil {
		snapshots := policyResolver.Snapshot()
		resolvedByProfile = make(map[string]int, len(snapshots))
		for _, snap := range snapshots {
			resolvedByProfile[snap.Name] = len(snap.ResolvedCIDRs)
		}
	}

	// Calculate column widths.
	nameW, matchW, viaW := 16, 28, 14
	for _, r := range rules {
		if l := len(r.Name); l+2 > nameW {
			nameW = l + 2
		}
		m := summarizeMatch(&r)
		if l := len(m); l+2 > matchW {
			matchW = l + 2
		}
		if l := len(r.Via); l+2 > viaW {
			viaW = l + 2
		}
	}
	if matchW > 40 {
		matchW = 40
	}

	orderW, orderHdr := 0, ""
	if withOrder {
		orderW = max(14, utf8.RuneCountInString(ct("cli.profile.priority.by_position"))+2)
		orderHdr = styleMapHdr.Render(padRunes(ct("cli.policy.list.col.order"), orderW)) + " "
	}

	sep := styleDim.Render("  " + strings.Repeat("─", nameW+matchW+viaW+orderW+12))
	fmt.Printf("  %s %s %s %s%s\n",
		styleMapHdr.Render(pad("POLICY", nameW)),
		styleMapHdr.Render(pad("MATCH", matchW)),
		styleMapHdr.Render(pad("VIA", viaW)),
		orderHdr,
		styleMapHdr.Render("RESOLVED"))
	fmt.Println(sep)

	for _, r := range rules {
		match := summarizeMatch(&r)
		if len(match) > matchW {
			match = match[:matchW-1] + "…"
		}

		resolved := styleDim.Render("—")
		if resolvedByProfile != nil {
			if n, ok := resolvedByProfile[r.Via]; ok && n > 0 {
				resolved = styleMapResolved.Render(fmt.Sprintf("%d IPs", n))
			}
		}

		// Color match column: domains=blue, IPs=orange, mixed=plain
		matchPadded := pad(match, matchW)
		var matchColored string
		if len(r.Match.Domains) > 0 && len(r.Match.IPRanges) == 0 {
			matchColored = styleMapDomain.Render(matchPadded)
		} else if len(r.Match.IPRanges) > 0 && len(r.Match.Domains) == 0 {
			matchColored = styleMapIP.Render(matchPadded)
		} else {
			matchColored = matchPadded
		}

		order := ""
		if withOrder {
			p := profilePriority(r.Via)
			cell := padRunes(priorityText(p), orderW)
			if p > 0 {
				order = styleBright.Render(cell) + " "
			} else {
				order = styleDim.Render(cell) + " "
			}
		}

		fmt.Printf("  %s %s %s %s%s\n",
			styleMapName.Render(pad(r.Name, nameW)),
			matchColored,
			styleMapVia.Render(pad(r.Via, viaW)),
			order,
			resolved)
	}
	fmt.Println(sep)
	if withOrder {
		fmt.Println("  " + styleDim.Render(ct("cli.policy.list.order_hint")))
	}
}

func buildMapReport(args []string) mapReportJSON {
	out := mapReportJSON{Rules: []mapRuleJSON{}}
	if len(args) > 0 {
		ex := engine.ExplainTarget(args[0])
		out.Target = &ex
	}

	rules := engine.Rules()
	if len(rules) == 0 {
		return out
	}

	resolvedByProfile := map[string]int{}
	if policyResolver != nil {
		for _, snap := range policyResolver.Snapshot() {
			resolvedByProfile[snap.Name] = len(snap.ResolvedCIDRs)
		}
	}

	out.Rules = make([]mapRuleJSON, 0, len(rules))
	for _, r := range rules {
		row := mapRuleJSON{
			Name:     r.Name,
			Via:      r.Via,
			Apps:     append([]string(nil), r.Match.Apps...),
			Domains:  append([]string(nil), r.Match.Domains...),
			IPRanges: make([]string, 0, len(r.Match.IPRanges)),
		}
		for _, p := range r.Match.IPRanges {
			row.IPRanges = append(row.IPRanges, p.String())
		}
		row.ResolvedIPs = resolvedByProfile[r.Via]
		row.Priority = profilePriority(r.Via)
		out.Rules = append(out.Rules, row)
	}

	return out
}

// pad right-pads s with spaces to width w (display-safe, no ANSI).
func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func summarizeMatch(r *policy.Rule) string {
	var parts []string
	for _, a := range r.Match.Apps {
		parts = append(parts, "app:"+a)
	}
	for _, d := range r.Match.Domains {
		parts = append(parts, d)
	}
	for _, n := range r.Match.IPRanges {
		parts = append(parts, n.String())
	}
	return strings.Join(parts, ", ")
}

// padRunes right-pads s to w runes ("por posición" has a multi-byte rune).
func padRunes(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

// profilePriority is the priority of the profile named via (0 = unset or
// unknown: the rule's position decides).
func profilePriority(via string) int {
	if cfg == nil {
		return 0
	}
	return cfg.VPNs[via].Priority
}
