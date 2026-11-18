package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Deprecated commands and flags keep working, hidden from --help and the
// generated docs, and print one translated line pointing at the new name.
// cobra's Command.Deprecated / pflag's MarkDeprecated print a fixed English
// sentence ("Command %q is deprecated, …", "Flag --%s has been deprecated,
// …"; cobra v1.8.0 command.go, pflag v1.0.5 flag.go), so they are not used:
// every user-visible string goes through internal/i18n (AGENTS.md).

// annotationDeprecated marks a hidden command that is a deprecated alias
// (its value is the replacement), so it still counts as the user's own
// command (session "last use") unlike hidden plumbing such as daemon.
const annotationDeprecated = "kongtrol.deprecated"

// warnDeprecated prints "<old> is deprecated: use <replacement>" on stderr,
// so --json output on stdout stays parseable.
func warnDeprecated(old, replacement string) {
	if outputQuiet {
		return
	}
	fmt.Fprintln(os.Stderr, tuiWarn(cf("cli.deprecated", old, replacement)))
}

// deprecatedCommand returns a hidden command use that prints the notice
// (old → replacement) and then runs run.
func deprecatedCommand(use, old, replacement string, args cobra.PositionalArgs, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:         use,
		Hidden:      true,
		Args:        args,
		Annotations: map[string]string{annotationDeprecated: replacement},
		RunE: func(c *cobra.Command, a []string) error {
			warnDeprecated(old, replacement)
			return run(c, a)
		},
	}
}
