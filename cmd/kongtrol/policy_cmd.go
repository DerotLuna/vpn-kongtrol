package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

var policyCmd = &cobra.Command{
	Use: "policy",
}

// policy list (was kongtrol map): every policy with its VPN and resolved IPs.
var policyListCmd = &cobra.Command{
	Use:  "list",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if engine == nil {
			return fmt.Errorf("%s", ct("cli.policy.engine_not_loaded"))
		}
		if outputJSON {
			return emitJSON(buildMapReport(nil))
		}
		printTrafficMap(true)
		return nil
	},
}

// policy show (was policy explain and kongtrol map <destino>): which policy
// and VPN a destination or app uses, and why.
var policyShowCmd = &cobra.Command{
	Use:  "show <target|app:exe>",
	Args: cobra.ExactArgs(1),
	RunE: runPolicyShow,
}

// policyExplainCmd is the deprecated name of policy show.
var policyExplainCmd = deprecatedCommand("explain <target|app:exe>", "kongtrol policy explain", "kongtrol policy show", cobra.ExactArgs(1), runPolicyShow)

func runPolicyShow(cmd *cobra.Command, args []string) error {
	if engine == nil {
		return fmt.Errorf("%s", ct("cli.policy.engine_not_loaded"))
	}

	target := strings.TrimSpace(args[0])
	ex := engine.ExplainTarget(target)
	if outputJSON {
		return emitJSON(ex)
	}
	if ex.Matched {
		fmt.Println(tuiOK(styleMapName.Render(target) + "  →  " + styleMapVia.Render(ex.Via)))
		if ex.RuleName != "" {
			fmt.Println("  " + styleDim.Render(ct("cli.policy.rule_label")) + styleBright.Render(ex.RuleName))
		}
		if ex.Reason != "" {
			fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+policyReasonText(ex)))
		}
		return nil
	}

	fmt.Println(tuiWarn(styleMapName.Render(target) + "  →  " + styleDim.Render(ct("cli.policy.default_route_short"))))
	if ex.Reason != "" {
		fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+policyReasonText(ex)))
	}
	return nil
}

var policyTestCmd = &cobra.Command{
	Use:  "test <targets-file>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if engine == nil {
			return fmt.Errorf("%s", ct("cli.policy.engine_not_loaded"))
		}
		targets, err := readPolicyTargets(args[0])
		if err != nil {
			return err
		}
		results := make([]policy.ExplainResult, 0, len(targets))
		for _, t := range targets {
			results = append(results, engine.ExplainTarget(t))
		}
		if outputJSON {
			return emitJSON(results)
		}
		for _, ex := range results {
			target := ex.Target
			if ex.Matched {
				fmt.Println(tuiOK(styleMapName.Render(target) + "  →  " + styleMapVia.Render(ex.Via)))
			} else {
				fmt.Println(tuiWarn(styleMapName.Render(target) + "  →  " + styleDim.Render(ct("cli.policy.default_route_short"))))
			}
			if ex.Reason != "" {
				fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+policyReasonText(ex)))
			}
		}
		return nil
	},
}

func readPolicyTargets(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ct("cli.policy.test.read_error"), err)
	}
	defer f.Close()
	out := []string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", ct("cli.policy.test.read_error"), err)
	}
	return out, nil
}

func init() {
	policyCmd.AddCommand(policyListCmd, policyShowCmd, policyTestCmd, policyExplainCmd)
	rootCmd.AddCommand(policyCmd)
}
