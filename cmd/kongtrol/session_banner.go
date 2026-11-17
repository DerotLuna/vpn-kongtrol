package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func prepareSessionBanner(now time.Time, cmd *cobra.Command) (greeting, lastUse string) {
	path := sessionStatePath()
	prev, err := loadSessionState(path)
	if err == nil {
		lastAt := prev.LastCommandAt
		if lastAt.IsZero() {
			lastAt = prev.LastLoginAt
		}
		if !lastAt.IsZero() {
			if strings.TrimSpace(prev.LastCommand) != "" {
				lastUse = cf("cli.session.last_use", formatLocalDate(lastAt.Local()), prev.LastCommand)
			} else {
				lastUse = cf("cli.session.last_use_simple", formatLocalDate(lastAt.Local()))
			}
		}
	}

	greeting = cf(statusGreetingKeyForHour(now.Hour()), resolveSystemUserName())
	if isInternalCommand(cmd) {
		// Hidden plumbing (daemon, helper run…) is launched by kongtrol
		// itself, not typed by the user — keep the last real command.
		return greeting, lastUse
	}
	_ = saveSessionState(path, cliSessionState{
		LastCommandAt: now,
		LastCommand:   cmd.CommandPath(),
		LastLoginAt:   now, // legacy compatibility
	})
	return greeting, lastUse
}

func statusGreetingKeyForHour(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "cli.status.greeting.morning"
	case hour >= 12 && hour < 20:
		return "cli.status.greeting.afternoon"
	default:
		return "cli.status.greeting.night"
	}
}

func printSessionBanner() {
	if sessionGreetingLine != "" {
		fmt.Println("  " + styleDim.Render(sessionGreetingLine))
	}
	if sessionLastUseLine != "" {
		fmt.Println("  " + styleDim.Render(sessionLastUseLine))
	}
	if sessionGreetingLine != "" || sessionLastUseLine != "" {
		fmt.Println()
	}
}

// isInternalCommand reports whether cmd (or any parent) is a hidden
// internal command that must not show up as the user's "last use".
func isInternalCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Hidden {
			return true
		}
	}
	return false
}
