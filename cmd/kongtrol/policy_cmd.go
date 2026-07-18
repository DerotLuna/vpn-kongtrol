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
	Use:   "policy",
	Short: "Policy tooling and diagnostics",
}

var policyExplainCmd = &cobra.Command{
	Use:   "explain <target|app:exe>",
	Short: "Explain which policy matched and why",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
				fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
			}
			return nil
		}

		fmt.Println(tuiWarn(styleMapName.Render(target) + "  →  " + styleDim.Render(ct("cli.policy.default_route_short"))))
		if ex.Reason != "" {
			fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
		}
		return nil
	},
}

var policyTestCmd = &cobra.Command{
	Use:   "test <targets-file>",
	Short: "Test policies against a target list",
	Args:  cobra.ExactArgs(1),
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
				fmt.Println("  " + styleDim.Render(ct("cli.policy.why_label")+ex.Reason))
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
	policyCmd.Short = ct("cli.policy.short")
	policyExplainCmd.Short = ct("cli.policy.explain.short")
	policyTestCmd.Short = ct("cli.policy.test.short")
	policyExplainCmd.Example = ct("cli.policy.explain.examples")
	policyTestCmd.Example = ct("cli.policy.test.examples")
	policyCmd.AddCommand(policyExplainCmd, policyTestCmd)
	rootCmd.AddCommand(policyCmd)
}
