package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// ── check ─────────────────────────────────────────────────────────────────────

// checkCmd looks up the public IP once, on demand (and compares it with
// integrity_check.expected_ips). The lookup is opt-in: while it is off no
// request is made and the command says how to turn it on.
var checkCmd = &cobra.Command{
	Use: "check",
	RunE: func(cmd *cobra.Command, args []string) error {
		var spin *spinner
		if !outputJSON {
			spin = newSpinner(ct("cli.check.spinner"))
			spin.Start()
		}
		result, host, err := runPublicIPCheck()
		if spin != nil {
			spin.Stop()
		}
		if err != nil {
			return err
		}
		if result.Disabled {
			if outputJSON {
				_ = emitJSON(struct {
					Enabled bool `json:"public_ip_check_enabled"`
				}{false})
			} else {
				fmt.Println(tuiWarn(ct("cli.check.disabled")))
				fmt.Println(tuiInfo(paint(cDim, ct("cli.check.enable_how"))))
			}
			return alreadyReported(errors.New(ct("cli.check.disabled")))
		}
		if outputJSON {
			if err := emitJSON(struct {
				Leak     bool   `json:"leak"`
				Reason   string `json:"reason,omitempty"`
				PublicIP string `json:"public_ip,omitempty"`
				Service  string `json:"service,omitempty"`
			}{
				Leak:     result.HasLeak,
				Reason:   result.Reason,
				PublicIP: result.PublicIP,
				Service:  host,
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
			fmt.Println(tuiInfo(paint(cDim, ct("cli.check.leak_hint"))))
			return alreadyReported(fmt.Errorf("%s", ct("cli.check.leak_detected")))
		}
		if result.PublicIP == "" {
			fmt.Println(tuiErr(paintBold(cBright, cf("cli.check.failed", host)) + "  " + paint(cDim, result.Reason)))
			fmt.Println(tuiInfo(paint(cDim, ct("cli.check.failed_hint"))))
			return alreadyReported(errors.New(cf("cli.check.failed", host)))
		}
		fmt.Println(tuiOK(paintBold(cBright, ct("cli.check.no_leak")) +
			"  " + paint(cDim, ct("cli.check.public_ip")) + styleStatusIP.Render(result.PublicIP)))
		fmt.Println(tuiInfo(paint(cDim, cf("cli.public_ip.notice", host))))
		return nil
	},
}

// runPublicIPCheck asks the running daemon (so every client sees the new
// result) or, without one, checks in this process.
func runPublicIPCheck() (security.LeakResult, string, error) {
	if base := daemonAPIBase(); probeDaemonAPI(base) {
		st, err := daemonClient(base).CheckPublicIP()
		var re *daemonctl.RemoteError
		switch {
		case errors.As(err, &re) && re.Key == "publicip.err.disabled":
			return security.LeakResult{Disabled: true}, st.Host, nil
		case err != nil:
			return security.LeakResult{}, "", err
		}
		res := security.LeakResult{}
		if st.Check != nil {
			res = security.LeakResult{
				PublicIP: st.Check.PublicIP, HasLeak: st.Check.HasLeak,
				Reason: st.Check.Reason, CheckedAt: st.Check.CheckedAt,
			}
		}
		return res, st.Host, nil
	}
	if leak == nil {
		return security.LeakResult{Disabled: true}, "", nil
	}
	return leak.CheckNow(), leak.Host(), nil
}
