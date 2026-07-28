package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ── check ─────────────────────────────────────────────────────────────────────

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run integrity and leak tests immediately",
	RunE: func(cmd *cobra.Command, args []string) error {
		if leak == nil {
			return fmt.Errorf("%s", ct("cli.check.leak_not_initialized"))
		}
		var spin *spinner
		if !outputJSON {
			spin = newSpinner(ct("cli.check.spinner"))
			spin.Start()
		}
		result := leak.CheckNow()
		if spin != nil {
			spin.Stop()
		}
		if outputJSON {
			if err := emitJSON(struct {
				Leak     bool   `json:"leak"`
				Reason   string `json:"reason,omitempty"`
				PublicIP string `json:"public_ip,omitempty"`
			}{
				Leak:     result.HasLeak,
				Reason:   result.Reason,
				PublicIP: result.PublicIP,
			}); err != nil {
				return err
			}
			if result.HasLeak {
				return fmt.Errorf("%s", ct("cli.check.leak_detected"))
			}
			return nil
		}
		if result.HasLeak {
			fmt.Println(tuiErr(paintBold(cBright, ct("cli.check.leak_detected")) + "  " + paint(cDim, result.Reason)))
			return fmt.Errorf("%s", ct("cli.check.leak_detected"))
		}
		fmt.Println(tuiOK(paintBold(cBright, ct("cli.check.no_leak")) +
			"  " + paint(cDim, ct("cli.check.public_ip")) + styleStatusIP.Render(result.PublicIP)))
		return nil
	},
}
