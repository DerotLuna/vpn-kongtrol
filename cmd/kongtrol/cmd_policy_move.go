package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// policy move: every rule of one profile goes through another, in one
// validated save (ADR-0104). The CLI side of the Add sheet's "Pasar reglas"
// after a provider-app profile is added: with a daemon through
// POST /api/v1/policies/move, otherwise the same configedit edit on
// kongtrol.yaml.

var policyMoveCmd = &cobra.Command{
	Use:  "move <from> <to>",
	Args: cobra.ExactArgs(2),
	RunE: runPolicyMove,
}

func runPolicyMove(_ *cobra.Command, args []string) error {
	from, to := strings.TrimSpace(args[0]), strings.TrimSpace(args[1])
	moved, viaDaemon, err := movePolicies(from, to)
	if err != nil {
		return err
	}
	if outputJSON {
		return emitJSON(map[string]any{"from": from, "to": to, "moved": nonNilStrings(moved), "via_daemon": viaDaemon})
	}
	if !outputQuiet {
		if len(moved) == 0 {
			fmt.Println(tuiInfo(cf("cli.policy.move.none", from)))
			return nil
		}
		fmt.Println(tuiOK(movedLine(len(moved), to)))
		fmt.Println(tuiInfo(ct("cli.policy.move.next")))
	}
	return nil
}

// movedLine is "Listo: N reglas ahora usan P." (singular for one).
func movedLine(n int, to string) string {
	return i18n.N(cliLang(), "cli.policy.move.done", n, to)
}

// movePolicies moves the rules through the running daemon, or in
// kongtrol.yaml.
func movePolicies(from, to string) (moved []string, viaDaemon bool, err error) {
	if base := daemonAPIBase(); daemonProbeFn(base) {
		moved, err := configClient(base).MovePolicies(from, to)
		return moved, true, remoteUserErr(err)
	}
	err = editConfigOffline(func(c *config.Config) error {
		var err error
		moved, err = configedit.MovePolicies(c, from, to)
		return err
	})
	return moved, false, err
}

// offerRuleMove runs after a provider-app profile (type typ) was created:
// for each direct profile made from that provider's files that has rules,
// it asks whether to move them to the new profile. Only on an interactive
// terminal and never answered by --yes (moving rules is its own choice);
// otherwise it prints the command that does it.
func offerRuleMove(name, typ string) {
	if outputJSON || outputQuiet {
		return
	}
	c, err := config.Load(activeCfgPath)
	if err != nil {
		return
	}
	label := providerAppLabels[typ]
	if label == "" {
		label = typ
	}
	for _, cand := range app.RuleMoveCandidates(c, typ) {
		fmt.Println(tuiInfo(i18n.N(cliLang(), "cli.policy.move_offer.body", cand.Rules, cand.Name, name, label)))
		if !confirmAction(cf("cli.policy.move_offer.ask", cand.Name, name), false, true) {
			fmt.Println(tuiInfo(cf("cli.policy.move_offer.hint", cand.Name, name)))
			continue
		}
		moved, _, err := movePolicies(cand.Name, name)
		if err != nil {
			fmt.Println(tuiWarn(err.Error()))
			fmt.Println(tuiInfo(cf("cli.policy.move_offer.hint", cand.Name, name)))
			continue
		}
		fmt.Println(tuiOK(movedLine(len(moved), name)))
	}
}

func init() {
	policyCmd.AddCommand(policyMoveCmd)
}
