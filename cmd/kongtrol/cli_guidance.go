package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// ── already-reported errors ──────────────────────────────────────────────────

// reportedError marks an error whose message the command already printed
// (with its own "what to do" line), so main exits non-zero without
// printing it a second time.
type reportedError struct{ err error }

func (e reportedError) Error() string { return e.err.Error() }
func (e reportedError) Unwrap() error { return e.err }

// alreadyReported wraps err so main doesn't print it again.
func alreadyReported(err error) error {
	if err == nil {
		return nil
	}
	return reportedError{err: err}
}

func isReported(err error) bool {
	var r reportedError
	return errors.As(err, &r)
}

// ── commands that don't need kongtrol.yaml ───────────────────────────────────

const annotationPrefsOnly = "kongtrol.prefs-only"

func init() {
	for _, c := range []*cobra.Command{
		configFavListCmd, configFavAddCmd, configFavRemoveCmd,
		configDefaultGroupCmd, configDefaultsShowCmd, configDefaultsSetGroupCmd,
		configLangCmd,
		configDashboardShowCmd, configDashboardSetPortCmd, configDashboardSetBindCmd, configDashboardResetCmd,
	} {
		if c.Annotations == nil {
			c.Annotations = map[string]string{}
		}
		c.Annotations[annotationPrefsOnly] = "true"
	}
}

// prefsOnlyCommand reports whether cmd only touches the machine-local
// preferences file, so it works before `kongtrol init` created a config.
func prefsOnlyCommand(cmd *cobra.Command) bool {
	return cmd.Annotations[annotationPrefsOnly] == "true"
}

// ── profile names ────────────────────────────────────────────────────────────

// configuredProfileNames returns the sorted profile names of the loaded config.
func configuredProfileNames() []string {
	if cfg == nil {
		return nil
	}
	names := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// validateProfileNames checks every name against the config before any
// work starts, suggesting the closest existing profile.
func validateProfileNames(names []string) error {
	if cfg == nil {
		return nil
	}
	known := configuredProfileNames()
	for _, name := range names {
		if _, ok := cfg.VPNs[name]; ok {
			continue
		}
		list := strings.Join(known, ", ")
		if s := closestName(name, known); s != "" {
			return fmt.Errorf("%s", cf("cli.error.unknown_profile_suggest", name, s, list))
		}
		return fmt.Errorf("%s", cf("cli.error.no_such_profile", name, list))
	}
	return nil
}

// closestName returns the candidate within a small edit distance of name.
func closestName(name string, candidates []string) string {
	best, bestDist := "", 3
	lower := strings.ToLower(name)
	for _, c := range candidates {
		cl := strings.ToLower(c)
		d := editDistance(lower, cl)
		if strings.HasPrefix(cl, lower) || strings.HasPrefix(lower, cl) {
			d = min(d, 1)
		}
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

// specifyProfileError asks which VPN to use, with an example and the list.
func specifyProfileError(verb string) error {
	names := configuredProfileNames()
	example := ct("cli.placeholder.profile")
	if len(names) > 0 {
		example = names[0]
	}
	return fmt.Errorf("%s", cf("cli.error.specify_profile_or_group", verb, example, strings.Join(names, ", ")))
}

// ── cobra usage errors ───────────────────────────────────────────────────────

var (
	reUnknownFlag    = regexp.MustCompile(`^unknown (?:shorthand )?flag: (.+)$`)
	reFlagNeedsArg   = regexp.MustCompile(`^flag needs an argument: (.+)$`)
	reInvalidFlagArg = regexp.MustCompile(`^invalid argument "(.*)" for "(.+?)" flag`)
	reUnknownCommand = regexp.MustCompile(`^unknown command "([^"]+)"`)
	reDidYouMean     = regexp.MustCompile(`(?s)Did you mean this\?\s+(\S+)`)
	reArgsCount      = regexp.MustCompile(`^(accepts \d+ arg|accepts at most|accepts between|requires at least|received \d+)`)
)

// flagError translates cobra/pflag flag errors into one guided line.
func flagError(cmd *cobra.Command, err error) error {
	msg := err.Error()
	detail := msg
	switch {
	case reUnknownFlag.MatchString(msg):
		detail = cf("cli.flag.unknown", reUnknownFlag.FindStringSubmatch(msg)[1])
	case reFlagNeedsArg.MatchString(msg):
		detail = cf("cli.flag.needs_value", reFlagNeedsArg.FindStringSubmatch(msg)[1])
	case reInvalidFlagArg.MatchString(msg):
		m := reInvalidFlagArg.FindStringSubmatch(msg)
		detail = cf("cli.flag.invalid_value", m[1], m[2])
	}
	return fmt.Errorf("%s", cf("cli.flag.error", detail, cmd.CommandPath()))
}

// userFacingError renders the error main prints, translating cobra's
// common English usage errors.
func userFacingError(cmd *cobra.Command, err error) string {
	msg := err.Error()
	if m := reUnknownCommand.FindStringSubmatch(msg); m != nil {
		if s := reDidYouMean.FindStringSubmatch(msg); s != nil {
			return cf("cli.error.unknown_command_suggest", m[1], s[1])
		}
		return cf("cli.error.unknown_command", m[1])
	}
	if reArgsCount.MatchString(msg) {
		path := "kongtrol"
		if cmd != nil {
			path = cmd.CommandPath()
		}
		return cf("cli.error.bad_args", path)
	}
	return msg
}

// ── rendering helpers ────────────────────────────────────────────────────────

// policyReasonText renders a policy explanation in the CLI language.
func policyReasonText(ex policy.ExplainResult) string {
	if ex.ReasonCode == "" {
		return ex.Reason
	}
	args := make([]any, len(ex.ReasonArgs))
	for i, a := range ex.ReasonArgs {
		args[i] = a
	}
	return cf("policy.reason."+ex.ReasonCode, args...)
}

// nothingConnectedHint is the next-step line when no tunnel is up.
func nothingConnectedHint(rows []tunnelRow) string {
	for _, r := range rows {
		if r.Status == vpn.StatusConnected || r.Status == vpn.StatusConnecting {
			return ""
		}
	}
	example := ct("cli.placeholder.profile")
	if len(rows) > 0 {
		example = rows[0].Name
	}
	return tuiInfo(cf("cli.status.nothing_connected", example))
}

// formatLocalDate formats t with day/month names in the CLI language.
func formatLocalDate(t time.Time) string {
	days := strings.Split(ct("date.days"), ",")
	months := strings.Split(ct("date.months"), ",")
	day, month := t.Weekday().String()[:3], t.Month().String()[:3]
	if len(days) == 7 {
		day = days[t.Weekday()]
	}
	if len(months) == 12 {
		month = months[t.Month()-1]
	}
	return cf("date.format", day, t.Day(), month, t.Year(), t.Format("15:04"))
}
