package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
)

// profile priority <name> [N|none]: shows or changes the number that orders
// this profile's rules against other rules that cover the same site or the
// same IP range (ADR-0108). Same shape as `policy state <name> [on|off]`;
// the panel's Perfiles editor has the same field. 0 or "none" clears it:
// the rule's position decides.
var profilePriorityCmd = &cobra.Command{
	Use:  "priority <name> [N|none]",
	Args: cobra.RangeArgs(1, 2),
	RunE: runProfilePriority,
}

// profile list: every profile with its type and order (priority or "by
// position"), like the panel's Perfiles list.
var profileListCmd = &cobra.Command{
	Use:  "list",
	Args: cobra.NoArgs,
	RunE: runProfileList,
}

// profilePriorityJSON is the --json answer of profile priority and one row
// of profile list. Priority 0 = unset (the rule's position decides).
type profilePriorityJSON struct {
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Priority   int    `json:"priority"`
	ByPosition bool   `json:"by_position"`
}

func priorityRow(name string, vc config.VPNConfig) profilePriorityJSON {
	return profilePriorityJSON{Name: name, Type: vc.Type, Priority: vc.Priority, ByPosition: vc.Priority == 0}
}

// parsePriority reads N, 0 or none (case-insensitive). Out of range and
// non-numbers get the same translated error as the config check.
func parsePriority(name, raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(raw, "none") {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > config.MaxPriority {
		return 0, localizeConfigErr(&config.ValidationError{Key: "config.err.profile_priority", Args: []any{name, raw, config.MaxPriority}})
	}
	return n, nil
}

// priorityText is "Prioridad N" or "por posición", as the panel shows it.
func priorityText(p int) string {
	if p > 0 {
		return cf("cli.profile.priority.value", p)
	}
	return ct("cli.profile.priority.by_position")
}

func runProfilePriority(_ *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	var vc config.VPNConfig
	ok := false
	if cfg != nil {
		vc, ok = cfg.VPNs[name]
	}
	if !ok {
		return localizeConfigErr(&config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
	}
	if len(args) == 2 {
		p, err := parsePriority(name, args[1])
		if err != nil {
			return err
		}
		if base := daemonAPIBase(); daemonProbeFn(base) {
			if err := configClient(base).UpdateVPN(name, configedit.VPNProfile{Priority: &p}); err != nil {
				return remoteUserErr(err)
			}
		} else if err := editConfigOffline(func(c *config.Config) error {
			profile, ok := c.VPNs[name]
			if !ok {
				return &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}}
			}
			profile.Priority = p
			c.VPNs[name] = profile
			return nil
		}); err != nil {
			return err
		}
		vc.Priority = p
		cfg.VPNs[name] = vc
		if outputJSON {
			return emitJSON(priorityRow(name, vc))
		}
		say(tuiOK(cf("cli.profile.priority.saved", name, priorityText(p))))
		say(tuiInfo(ct("cli.profile.priority.next")))
		return nil
	}
	if outputJSON {
		return emitJSON(priorityRow(name, vc))
	}
	say(tuiInfo(cf("cli.profile.priority.show", name, priorityText(vc.Priority))))
	say("  " + styleDim.Render(ct("cli.profile.priority.hint")))
	return nil
}

func runProfileList(_ *cobra.Command, _ []string) error {
	var vpns map[string]config.VPNConfig
	if cfg != nil {
		vpns = cfg.VPNs
	}
	names := make([]string, 0, len(vpns))
	for n := range vpns {
		names = append(names, n)
	}
	sort.Strings(names)
	rows := make([]profilePriorityJSON, 0, len(names))
	for _, n := range names {
		rows = append(rows, priorityRow(n, vpns[n]))
	}
	if outputJSON {
		return emitJSON(struct {
			Profiles []profilePriorityJSON `json:"profiles"`
		}{rows})
	}
	if outputQuiet {
		return nil
	}
	if len(rows) == 0 {
		fmt.Println("  " + styleDim.Render(ct("cli.profile.list.none")))
		return nil
	}
	nameW, typeW := 12, 10
	for _, r := range rows {
		nameW = max(nameW, len(r.Name)+2)
		typeW = max(typeW, len(r.Type)+2)
	}
	fmt.Printf("  %s %s %s\n",
		styleMapHdr.Render(pad(ct("cli.profile.list.col.name"), nameW)),
		styleMapHdr.Render(pad(ct("cli.profile.list.col.type"), typeW)),
		styleMapHdr.Render(ct("cli.profile.list.col.order")))
	for _, r := range rows {
		order := styleDim.Render(priorityText(r.Priority))
		if r.Priority > 0 {
			order = styleBright.Render(priorityText(r.Priority))
		}
		fmt.Printf("  %s %s %s\n", styleMapName.Render(pad(r.Name, nameW)), pad(r.Type, typeW), order)
	}
	fmt.Println()
	fmt.Println("  " + styleDim.Render(ct("cli.profile.priority.hint")))
	return nil
}

func init() {
	profileCmd.AddCommand(profilePriorityCmd, profileListCmd)
}
